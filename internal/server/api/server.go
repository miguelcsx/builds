// internal/server/api/server.go

package api

import (
	"context"
	"errors"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	buildv1 "builds/api/build"
	"builds/internal/server/db"
	models "builds/internal/server/db/models"
)

type Server struct {
	buildv1.UnimplementedBuildServiceServer
	db *db.Database
}

func NewServer(db *db.Database) *Server {
	return &Server{db: db}
}

func (s *Server) CreateBuild(ctx context.Context, req *buildv1.CreateBuildRequest) (*buildv1.Build, error) {
	if req.Build == nil {
		return nil, status.Error(codes.InvalidArgument, "build is required")
	}

	build := &models.Build{
		ID:        req.Build.Id,
		StartTime: req.Build.StartTime.AsTime(),
		EndTime:   req.Build.EndTime.AsTime(),
		Duration:  req.Build.Duration,
		Success:   req.Build.Success,
		Error:     req.Build.Error,
	}

	// Start a transaction
	err := s.db.DB.Transaction(func(tx *gorm.DB) error {
		// Create the build first
		if err := tx.Create(build).Error; err != nil {
			return err
		}

		// Create environment
		if req.Build.Environment != nil {
			if err := s.createEnvironment(tx, build.ID, req.Build.Environment); err != nil {
				return err
			}
		}

		// Create hardware
		if req.Build.Hardware != nil {
			if err := s.createHardware(tx, build.ID, req.Build.Hardware); err != nil {
				return err
			}
		}

		// Create compiler
		if req.Build.Compiler != nil {
			if err := s.createCompiler(tx, build.ID, req.Build.Compiler); err != nil {
				return err
			}
		}

		// Create command
		if req.Build.Command != nil {
			if err := s.createCommand(tx, build.ID, req.Build.Command); err != nil {
				return err
			}
		}

		// Create output
		if req.Build.Output != nil {
			if err := s.createOutput(tx, build.ID, req.Build.Output); err != nil {
				return err
			}
		}

		// Create resource usage
		if req.Build.ResourceUsage != nil {
			if err := s.createResourceUsage(tx, build.ID, req.Build.ResourceUsage); err != nil {
				return err
			}
		}

		// Create compiler remarks
		if len(req.Build.Remarks) > 0 {
			for _, remark := range req.Build.Remarks {
				compilerRemark := models.CompilerRemark{
					BuildID:  build.ID,
					Type:     remark.Type,
					Pass:     remark.Pass,
					Message:  remark.Message,
					Function: remark.Function,
					File:     remark.Location.File,
					Line:     remark.Location.Line,
					Column:   remark.Location.Column,
				}

				if err := tx.Create(&compilerRemark).Error; err != nil {
					return err
				}

				// Create remark arguments
				for _, arg := range remark.Args {
					remarkArg := models.RemarkArg{
						RemarkID:  compilerRemark.ID,
						StringVal: arg.StringVal,
						Callee:    arg.Callee,
						Reason:    arg.Reason,
					}

					if err := tx.Create(&remarkArg).Error; err != nil {
						return err
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Fetch the complete build with all relationships
	var completeBuild models.Build
	err = s.db.DB.
		Preload("Environment.Variables").
		Preload("Hardware.GPUs").
		Preload("Compiler.Options").
		Preload("Compiler.Optimizations").
		Preload("Compiler.Extensions").
		Preload("Command.Arguments").
		Preload("Output.Artifacts").
		Preload("CompilerRemarks.Args").
		Preload("ResourceUsage").
		Preload("Performance.Phases").
		First(&completeBuild, "id = ?", build.ID).Error

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return s.convertBuildToProto(&completeBuild), nil
}

func (s *Server) GetBuild(ctx context.Context, req *buildv1.GetBuildRequest) (*buildv1.Build, error) {
	var build models.Build

	err := s.db.DB.
		Preload("Environment.Variables").
		Preload("Hardware.GPUs").
		Preload("Compiler.Options").
		Preload("Compiler.Optimizations").
		Preload("Compiler.Extensions").
		Preload("Command.Arguments").
		Preload("Output.Artifacts").
		Preload("CompilerRemarks.Args").
		Preload("ResourceUsage").
		Preload("Performance.Phases").
		First(&build, "id = ?", req.Id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "build not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return s.convertBuildToProto(&build), nil
}

func (s *Server) ListBuilds(ctx context.Context, req *buildv1.ListBuildsRequest) (*buildv1.ListBuildsResponse, error) {
	builds, err := s.db.ListBuilds(int(req.PageSize), req.PageToken)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	response := &buildv1.ListBuildsResponse{
		Builds: make([]*buildv1.Build, len(builds)),
	}

	for i, build := range builds {
		response.Builds[i] = s.convertBuildToProto(&build)
	}

	return response, nil
}

func (s *Server) DeleteBuild(ctx context.Context, req *buildv1.DeleteBuildRequest) (*emptypb.Empty, error) {
	if err := s.db.DeleteBuild(req.Id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "build not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) StreamBuilds(req *buildv1.StreamBuildsRequest, stream buildv1.BuildService_StreamBuildsServer) error {
	ctx := stream.Context()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	lastTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			var builds []models.Build
			err := s.db.DB.
				Where("start_time > ?", lastTime).
				Order("start_time ASC").
				Find(&builds).Error

			if err != nil {
				return status.Error(codes.Internal, err.Error())
			}

			for _, build := range builds {
				if build.StartTime.After(lastTime) {
					lastTime = build.StartTime
				}
				if err := stream.Send(s.convertBuildToProto(&build)); err != nil {
					return err
				}
			}
		}
	}
}

// Helper functions for creating related entities
func (s *Server) createEnvironment(tx *gorm.DB, buildID string, env *buildv1.Environment) error {
	dbEnv := &models.Environment{
		BuildID:    buildID,
		OS:         env.Os,
		Arch:       env.Arch,
		WorkingDir: env.WorkingDir,
		Variables:  make([]models.EnvironmentVariable, 0, len(env.Variables)),
	}

	for k, v := range env.Variables {
		dbEnv.Variables = append(dbEnv.Variables, models.EnvironmentVariable{
			BuildID: buildID,
			Key:     k,
			Value:   v,
		})
	}

	return tx.Create(dbEnv).Error
}

func (s *Server) createHardware(tx *gorm.DB, buildID string, hw *buildv1.Hardware) error {
	dbHw := &models.Hardware{
		BuildID:    buildID,
		CPUModel:   hw.Cpu.Model,
		CPUFreq:    hw.Cpu.Frequency,
		CPUCores:   hw.Cpu.Cores,
		CPUThreads: hw.Cpu.Threads,
		CPUVendor:  hw.Cpu.Vendor,
		CacheSize:  hw.Cpu.CacheSize,
		MemTotal:   hw.Memory.Total,
		MemAvail:   hw.Memory.Available,
		MemUsed:    hw.Memory.Used,
		SwapTotal:  hw.Memory.SwapTotal,
		SwapFree:   hw.Memory.SwapFree,
		GPUs:       make([]models.GPU, len(hw.Gpus)),
	}

	for i, gpu := range hw.Gpus {
		dbHw.GPUs[i] = models.GPU{
			BuildID:     buildID,
			Model:       gpu.Model,
			Memory:      gpu.Memory,
			Driver:      gpu.Driver,
			ComputeCaps: gpu.ComputeCaps,
		}
	}

	return tx.Create(dbHw).Error
}

func (s *Server) createCompiler(tx *gorm.DB, buildID string, comp *buildv1.Compiler) error {
	dbComp := &models.Compiler{
		BuildID:         buildID,
		Name:            comp.Name,
		Version:         comp.Version,
		Target:          comp.Target,
		LanguageName:    comp.Language.Name,
		LanguageVersion: comp.Language.Version,
		LanguageSpec:    comp.Language.Specification,
		SupportsOpenMP:  comp.Features.SupportsOpenmp,
		SupportsGPU:     comp.Features.SupportsGpu,
		SupportsLTO:     comp.Features.SupportsLto,
		SupportsPGO:     comp.Features.SupportsPgo,
		Options:         make([]models.CompilerOption, len(comp.Options)),
		Optimizations:   make([]models.CompilerOptimization, 0),
		Extensions:      make([]models.CompilerExtension, len(comp.Features.Extensions)),
	}

	// Store options
	for i, opt := range comp.Options {
		dbComp.Options[i] = models.CompilerOption{
			BuildID: buildID,
			Option:  opt,
		}
	}

	// Store optimizations
	for name, enabled := range comp.Optimizations {
		dbComp.Optimizations = append(dbComp.Optimizations, models.CompilerOptimization{
			BuildID: buildID,
			Name:    name,
			Enabled: enabled,
		})
	}

	// Store extensions
	for i, ext := range comp.Features.Extensions {
		dbComp.Extensions[i] = models.CompilerExtension{
			BuildID:   buildID,
			Extension: ext,
		}
	}

	return tx.Create(dbComp).Error
}

func (s *Server) createCommand(tx *gorm.DB, buildID string, cmd *buildv1.Command) error {
	dbCmd := &models.Command{
		BuildID:    buildID,
		Executable: cmd.Executable,
		WorkingDir: cmd.WorkingDir,
		Arguments:  make([]models.CommandArgument, len(cmd.Arguments)),
	}

	for i, arg := range cmd.Arguments {
		dbCmd.Arguments[i] = models.CommandArgument{
			BuildID:  buildID,
			Position: i,
			Argument: arg,
		}
	}

	return tx.Create(dbCmd).Error
}

func (s *Server) createOutput(tx *gorm.DB, buildID string, output *buildv1.Output) error {
	dbOutput := &models.Output{
		BuildID:   buildID,
		Stdout:    output.Stdout,
		Stderr:    output.Stderr,
		ExitCode:  output.ExitCode,
		Artifacts: make([]models.Artifact, len(output.Artifacts)),
	}

	for i, artifact := range output.Artifacts {
		dbOutput.Artifacts[i] = models.Artifact{
			BuildID: buildID,
			Path:    artifact.Path,
			Type:    artifact.Type,
			Size:    artifact.Size,
			Hash:    artifact.Hash,
		}
	}

	return tx.Create(dbOutput).Error
}

func (s *Server) createResourceUsage(tx *gorm.DB, buildID string, usage *buildv1.ResourceUsage) error {
	dbUsage := &models.ResourceUsage{
		BuildID:    buildID,
		MaxMemory:  usage.MaxMemory,
		CPUTime:    usage.CpuTime,
		Threads:    usage.Threads,
		ReadBytes:  usage.Io.ReadBytes,
		WriteBytes: usage.Io.WriteBytes,
		ReadCount:  usage.Io.ReadCount,
		WriteCount: usage.Io.WriteCount,
	}

	return tx.Create(dbUsage).Error
}

// internal/server/api/server.go

func (s *Server) convertBuildToProto(build *models.Build) *buildv1.Build {
	pb := &buildv1.Build{
		Id:        build.ID,
		StartTime: timestamppb.New(build.StartTime),
		EndTime:   timestamppb.New(build.EndTime),
		Duration:  build.Duration,
		Success:   build.Success,
		Error:     build.Error,
		Environment: &buildv1.Environment{
			Os:         build.Environment.OS,
			Arch:       build.Environment.Arch,
			WorkingDir: build.Environment.WorkingDir,
			Variables:  make(map[string]string),
		},
		Hardware: &buildv1.Hardware{
			Cpu: &buildv1.CPU{
				Model:     build.Hardware.CPUModel,
				Vendor:    build.Hardware.CPUVendor,
				Cores:     build.Hardware.CPUCores,
				Threads:   build.Hardware.CPUThreads,
				Frequency: build.Hardware.CPUFreq,
				CacheSize: build.Hardware.CacheSize,
			},
			Memory: &buildv1.Memory{
				Total:     build.Hardware.MemTotal,
				Available: build.Hardware.MemAvail,
				Used:      build.Hardware.MemUsed,
				SwapTotal: build.Hardware.SwapTotal,
				SwapFree:  build.Hardware.SwapFree,
			},
			Gpus: make([]*buildv1.GPU, 0, len(build.Hardware.GPUs)),
		},
		Compiler: &buildv1.Compiler{
			Name:          build.Compiler.Name,
			Version:       build.Compiler.Version,
			Target:        build.Compiler.Target,
			Options:       make([]string, 0),
			Optimizations: make(map[string]bool),
			Flags:         make(map[string]string),
			Language: &buildv1.Language{
				Name:          build.Compiler.LanguageName,
				Version:       build.Compiler.LanguageVersion,
				Specification: build.Compiler.LanguageSpec,
			},
			Features: &buildv1.CompilerFeatures{
				SupportsOpenmp: build.Compiler.SupportsOpenMP,
				SupportsGpu:    build.Compiler.SupportsGPU,
				SupportsLto:    build.Compiler.SupportsLTO,
				SupportsPgo:    build.Compiler.SupportsPGO,
				Extensions:     make([]string, 0),
			},
		},
		Command: &buildv1.Command{
			Executable: build.Command.Executable,
			WorkingDir: build.Command.WorkingDir,
			Arguments:  make([]string, 0),
			Env:        make(map[string]string),
		},
		Output: &buildv1.Output{
			Stdout:    build.Output.Stdout,
			Stderr:    build.Output.Stderr,
			ExitCode:  build.Output.ExitCode,
			Artifacts: make([]*buildv1.Artifact, 0),
			Warnings:  make([]string, 0),
			Errors:    make([]string, 0),
		},
		ResourceUsage: &buildv1.ResourceUsage{
			MaxMemory: build.ResourceUsage.MaxMemory,
			CpuTime:   build.ResourceUsage.CPUTime,
			Threads:   build.ResourceUsage.Threads,
			Io: &buildv1.IOStats{
				ReadBytes:  build.ResourceUsage.ReadBytes,
				WriteBytes: build.ResourceUsage.WriteBytes,
				ReadCount:  build.ResourceUsage.ReadCount,
				WriteCount: build.ResourceUsage.WriteCount,
			},
		},
		Performance: &buildv1.Performance{
			CompileTime:  build.Performance.CompileTime,
			LinkTime:     build.Performance.LinkTime,
			OptimizeTime: build.Performance.OptimizeTime,
			Phases:       make(map[string]float64),
		},
		Remarks: make([]*buildv1.CompilerRemark, 0),
	}

	// Add environment variables
	for _, v := range build.Environment.Variables {
		pb.Environment.Variables[v.Key] = v.Value
	}

	// Add GPUs
	for _, gpu := range build.Hardware.GPUs {
		pb.Hardware.Gpus = append(pb.Hardware.Gpus, &buildv1.GPU{
			Model:       gpu.Model,
			Memory:      gpu.Memory,
			Driver:      gpu.Driver,
			ComputeCaps: gpu.ComputeCaps,
		})
	}

	// Add compiler options
	for _, opt := range build.Compiler.Options {
		pb.Compiler.Options = append(pb.Compiler.Options, opt.Option)
	}

	// Add compiler optimizations
	for _, opt := range build.Compiler.Optimizations {
		pb.Compiler.Optimizations[opt.Name] = opt.Enabled
	}

	// Add compiler extensions
	for _, ext := range build.Compiler.Extensions {
		pb.Compiler.Features.Extensions = append(pb.Compiler.Features.Extensions, ext.Extension)
	}

	// Add command arguments
	for _, arg := range build.Command.Arguments {
		pb.Command.Arguments = append(pb.Command.Arguments, arg.Argument)
	}

	// Add artifacts
	for _, artifact := range build.Output.Artifacts {
		pb.Output.Artifacts = append(pb.Output.Artifacts, &buildv1.Artifact{
			Path: artifact.Path,
			Type: artifact.Type,
			Size: artifact.Size,
			Hash: artifact.Hash,
		})
	}

	// Add performance phases
	for _, phase := range build.Performance.Phases {
		pb.Performance.Phases[phase.Phase] = phase.Duration
	}

	// Add compiler remarks
	for _, remark := range build.CompilerRemarks {
		pbRemark := &buildv1.CompilerRemark{
			Type:     remark.Type,
			Pass:     remark.Pass,
			Message:  remark.Message,
			Function: remark.Function,
			Location: &buildv1.Location{
				File:     remark.File,
				Line:     remark.Line,
				Column:   remark.Column,
				Function: remark.Function,
			},
			Args: make([]*buildv1.RemarkArg, 0),
		}

		for _, arg := range remark.Args {
			pbRemark.Args = append(pbRemark.Args, &buildv1.RemarkArg{
				StringVal: arg.StringVal,
				Callee:    arg.Callee,
				Reason:    arg.Reason,
			})
		}

		pb.Remarks = append(pb.Remarks, pbRemark)
	}

	return pb
}

func getOffset(pageToken string) int {
	if pageToken == "" {
		return 0
	}

	// Try to parse the token as an integer offset
	offset, err := strconv.Atoi(pageToken)
	if err != nil {
		return 0
	}

	// Ensure offset is non-negative
	if offset < 0 {
		return 0
	}

	// Optional: Add a maximum offset limit to prevent excessive queries
	const maxOffset = 10000
	if offset > maxOffset {
		return maxOffset
	}

	return offset
}
