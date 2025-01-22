// internal/server/db/database.go

package db

import (
	models "builds/internal/server/db/models"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type Database struct {
	DB *gorm.DB
}

func New(db *gorm.DB) *Database {
	return &Database{DB: db}
}

func (d *Database) Migrate() error {
	// The order is important here due to foreign key constraints
	modelsList := []interface{}{
		// Core models
		&models.Build{},
		&models.Environment{},
		&models.EnvironmentVariable{},
		&models.Hardware{},
		&models.GPU{},
		&models.Compiler{},
		&models.CompilerOption{},
		&models.CompilerOptimization{},
		&models.CompilerExtension{},
		&models.Command{},
		&models.CommandArgument{},
		&models.Output{},
		&models.Artifact{},
		&models.ResourceUsage{},
		&models.Performance{},
		&models.PerformancePhase{},

		// Remarks and related models
		&models.CompilerRemark{},
		&models.KernelInfo{},
		&models.MemoryAccess{},
	}

	// Create custom types first
	if err := d.createCustomTypes(); err != nil {
		return fmt.Errorf("failed to create custom types: %w", err)
	}

	// Migrate models
	for _, model := range modelsList {
		if err := d.DB.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	return nil
}

func (d *Database) CreateBuildWithRelations(build *models.Build) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		// Create the main build record
		if err := tx.Create(build).Error; err != nil {
			return fmt.Errorf("failed to create build: %w", err)
		}

		// Create Environment
		if build.Environment.BuildID != "" {
			if err := tx.Create(&build.Environment).Error; err != nil {
				return fmt.Errorf("failed to create environment: %w", err)
			}

			if len(build.Environment.Variables) > 0 {
				if err := tx.Create(&build.Environment.Variables).Error; err != nil {
					return fmt.Errorf("failed to create environment variables: %w", err)
				}
			}
		}

		// Create Hardware
		if build.Hardware.BuildID != "" {
			if err := tx.Create(&build.Hardware).Error; err != nil {
				return fmt.Errorf("failed to create hardware: %w", err)
			}

			if len(build.Hardware.GPUs) > 0 {
				if err := tx.Create(&build.Hardware.GPUs).Error; err != nil {
					return fmt.Errorf("failed to create GPUs: %w", err)
				}
			}
		}

		// Create Compiler
		if build.Compiler.BuildID != "" {
			if err := tx.Create(&build.Compiler).Error; err != nil {
				return fmt.Errorf("failed to create compiler: %w", err)
			}

			if len(build.Compiler.Options) > 0 {
				if err := tx.Create(&build.Compiler.Options).Error; err != nil {
					return fmt.Errorf("failed to create compiler options: %w", err)
				}
			}

			if len(build.Compiler.Optimizations) > 0 {
				if err := tx.Create(&build.Compiler.Optimizations).Error; err != nil {
					return fmt.Errorf("failed to create compiler optimizations: %w", err)
				}
			}

			if len(build.Compiler.Extensions) > 0 {
				if err := tx.Create(&build.Compiler.Extensions).Error; err != nil {
					return fmt.Errorf("failed to create compiler extensions: %w", err)
				}
			}
		}

		// Create Command
		if build.Command.BuildID != "" {
			if err := tx.Create(&build.Command).Error; err != nil {
				return fmt.Errorf("failed to create command: %w", err)
			}

			if len(build.Command.Arguments) > 0 {
				if err := tx.Create(&build.Command.Arguments).Error; err != nil {
					return fmt.Errorf("failed to create command arguments: %w", err)
				}
			}
		}

		// Create Output
		if build.Output.BuildID != "" {
			if err := tx.Create(&build.Output).Error; err != nil {
				return fmt.Errorf("failed to create output: %w", err)
			}

			if len(build.Output.Artifacts) > 0 {
				if err := tx.Create(&build.Output.Artifacts).Error; err != nil {
					return fmt.Errorf("failed to create artifacts: %w", err)
				}
			}
		}

		// Create Remarks
		if len(build.Remarks) > 0 {
			for _, remark := range build.Remarks {
				// Set the build ID for the remark
				remark.BuildID = build.ID

				if err := tx.Create(&remark).Error; err != nil {
					return fmt.Errorf("failed to create compiler remark: %w", err)
				}

				// Create kernel info if present
				if remark.KernelInfo != nil {
					remark.KernelInfo.RemarkID = remark.ID

					if err := tx.Create(remark.KernelInfo).Error; err != nil {
						return fmt.Errorf("failed to create kernel info: %w", err)
					}

					// Create memory accesses
					if len(remark.KernelInfo.MemoryAccesses) > 0 {
						for i := range remark.KernelInfo.MemoryAccesses {
							remark.KernelInfo.MemoryAccesses[i].KernelInfoID = remark.KernelInfo.ID
						}

						if err := tx.Create(&remark.KernelInfo.MemoryAccesses).Error; err != nil {
							return fmt.Errorf("failed to create memory accesses: %w", err)
						}
					}
				}
			}
		}

		// Create ResourceUsage
		if build.ResourceUsage.BuildID != "" {
			if err := tx.Create(&build.ResourceUsage).Error; err != nil {
				return fmt.Errorf("failed to create resource usage: %w", err)
			}
		}

		// Create Performance
		if build.Performance.BuildID != "" {
			if err := tx.Create(&build.Performance).Error; err != nil {
				return fmt.Errorf("failed to create performance: %w", err)
			}

			if len(build.Performance.Phases) > 0 {
				if err := tx.Create(&build.Performance.Phases).Error; err != nil {
					return fmt.Errorf("failed to create performance phases: %w", err)
				}
			}
		}

		return nil
	})
}

func (d *Database) GetBuildByID(id string) (*models.Build, error) {
	var build models.Build

	result := d.DB.
		Preload("Environment.Variables").
		Preload("Hardware.GPUs").
		Preload("Compiler.Options").
		Preload("Compiler.Optimizations").
		Preload("Compiler.Extensions").
		Preload("Command.Arguments").
		Preload("Output.Artifacts").
		Preload("ResourceUsage").
		Preload("Performance").
		Preload("Performance.Phases").
		First(&build, "id = ?", id)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get build: %w", result.Error)
	}

	// Load remarks separately to handle potential empty remarks
	var remarks []models.CompilerRemark
	if err := d.DB.
		Where("build_id = ?", build.ID).
		Preload("KernelInfo").
		Preload("KernelInfo.MemoryAccesses").
		Find(&remarks).Error; err != nil {
		return nil, fmt.Errorf("failed to load remarks: %w", err)
	}

	build.Remarks = remarks
	return &build, nil
}

func (d *Database) ListBuilds(pageSize int, lastID string) ([]models.Build, error) {
	var builds []models.Build

	query := d.DB.Model(&models.Build{}).Order("created_at DESC")

	if lastID != "" {
		var lastBuild models.Build
		if err := d.DB.First(&lastBuild, "id = ?", lastID).Error; err != nil {
			return nil, err
		}
		query = query.Where("created_at < ?", lastBuild.CreatedAt)
	}

	err := query.
		Preload("Environment").
		Preload("Hardware").
		Preload("Compiler").
		Preload("ResourceUsage").
		Limit(pageSize).
		Find(&builds).Error

	if err != nil {
		return nil, err
	}

	// Load remarks separately for each build
	for i := range builds {
		if err := d.DB.
			Where("build_id = ?", builds[i].ID).
			Preload("KernelInfo").
			Find(&builds[i].Remarks).Error; err != nil {
			return nil, fmt.Errorf("failed to load remarks for build %s: %w", builds[i].ID, err)
		}
	}

	return builds, nil
}

func (d *Database) DeleteBuild(id string) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		// Delete related records first to maintain referential integrity
		if err := tx.Where("build_id = ?", id).Delete(&models.CompilerRemark{}).Error; err != nil {
			return err
		}

		// Delete the build
		result := tx.Where("id = ?", id).Delete(&models.Build{})
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return nil
	})
}

func (d *Database) GetBuildsAfter(timestamp string) ([]models.Build, error) {
	var builds []models.Build

	err := d.DB.
		Where("created_at > ?", timestamp).
		Order("created_at ASC").
		Preload("Environment").
		Preload("Hardware").
		Preload("Compiler").
		Preload("Remarks").
		Preload("Remarks.KernelInfo").
		Find(&builds).Error

	if err != nil {
		return nil, err
	}

	return builds, nil
}

func (d *Database) createCustomTypes() error {
	// Create enums if needed
	type enumInfo struct {
		name       string
		values     []string
		defaultVal string
	}

	enums := []enumInfo{
		{
			name:       "remark_type",
			values:     []string{"optimization", "kernel", "analysis", "metric", "info"},
			defaultVal: "info",
		},
		{
			name:       "remark_pass",
			values:     []string{"vectorization", "inlining", "kernel-info", "size-info", "analysis"},
			defaultVal: "analysis",
		},
		{
			name:       "remark_status",
			values:     []string{"passed", "missed", "analysis"},
			defaultVal: "passed",
		},
	}

	for _, enum := range enums {
		// Check if type exists
		var exists bool
		err := d.DB.Raw(`
            SELECT EXISTS (
                SELECT 1 FROM pg_type t 
                JOIN pg_namespace n ON t.typnamespace = n.oid 
                WHERE t.typname = ? AND n.nspname = 'public'
            )`, enum.name).Scan(&exists).Error
		if err != nil {
			return fmt.Errorf("failed to check enum %s: %w", enum.name, err)
		}

		if !exists {
			sql := fmt.Sprintf(`DO $$ 
            BEGIN
                IF NOT EXISTS (SELECT 1 FROM pg_type t 
                    JOIN pg_namespace n ON t.typnamespace = n.oid 
                    WHERE t.typname = '%s' AND n.nspname = 'public') 
                THEN
                    CREATE TYPE %s AS ENUM ('%s');
                END IF;
            END $$;`, enum.name, enum.name, strings.Join(enum.values, "', '"))

			if err := d.DB.Exec(sql).Error; err != nil {
				return fmt.Errorf("failed to create enum %s: %w", enum.name, err)
			}
		}
	}

	return nil
}

// Ensure table consistency
func (d *Database) EnsureTables() error {
	// Check if KernelInfo table exists
	if !d.DB.Migrator().HasTable(&models.KernelInfo{}) {
		if err := d.DB.AutoMigrate(&models.KernelInfo{}); err != nil {
			return fmt.Errorf("failed to create kernel_infos table: %w", err)
		}
	}

	// Check if MemoryAccess table exists
	if !d.DB.Migrator().HasTable(&models.MemoryAccess{}) {
		if err := d.DB.AutoMigrate(&models.MemoryAccess{}); err != nil {
			return fmt.Errorf("failed to create memory_accesses table: %w", err)
		}
	}

	return nil
}
