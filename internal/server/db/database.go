// internal/server/db/database.go

package db

import (
	models "builds/internal/server/db/models"
	"fmt"

	"gorm.io/gorm"
)

type Database struct {
	DB *gorm.DB
}

func New(db *gorm.DB) *Database {
	return &Database{DB: db}
}

func (d *Database) Migrate() error {
	modelsList := []interface{}{
		&models.Build{},
		&models.Environment{},
		&models.EnvironmentVariable{},
		&models.Hardware{},
		&models.GPU{},
		&models.Compiler{},
		&models.CompilerOptimization{},
		&models.CompilerExtension{},
		&models.Command{},
		&models.CommandArgument{},
		&models.Output{},
		&models.Artifact{},
		&models.CompilerRemark{},
		&models.RemarkArg{},
		&models.ResourceUsage{},
		&models.Performance{},
		&models.PerformancePhase{},
	}

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

			// Create environment variables
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

			// Create GPUs
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

			// Create compiler options
			if len(build.Compiler.Options) > 0 {
				if err := tx.Create(&build.Compiler.Options).Error; err != nil {
					return fmt.Errorf("failed to create compiler options: %w", err)
				}
			}

			// Create compiler optimizations
			if len(build.Compiler.Optimizations) > 0 {
				if err := tx.Create(&build.Compiler.Optimizations).Error; err != nil {
					return fmt.Errorf("failed to create compiler optimizations: %w", err)
				}
			}

			// Create compiler extensions
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

			// Create command arguments
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

			// Create artifacts
			if len(build.Output.Artifacts) > 0 {
				if err := tx.Create(&build.Output.Artifacts).Error; err != nil {
					return fmt.Errorf("failed to create artifacts: %w", err)
				}
			}
		}

		// Create CompilerRemarks
		if len(build.CompilerRemarks) > 0 {
			for i := range build.CompilerRemarks {
				if err := tx.Create(&build.CompilerRemarks[i]).Error; err != nil {
					return fmt.Errorf("failed to create compiler remark: %w", err)
				}

				// Create remark arguments
				if len(build.CompilerRemarks[i].Args) > 0 {
					if err := tx.Create(&build.CompilerRemarks[i].Args).Error; err != nil {
						return fmt.Errorf("failed to create remark arguments: %w", err)
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

			// Create performance phases
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

	err := d.DB.
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
		First(&build, "id = ?", id).Error

	if err != nil {
		return nil, err
	}

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
		Limit(pageSize).
		Preload("Environment").
		Preload("Hardware").
		Preload("Compiler").
		Find(&builds).Error

	if err != nil {
		return nil, err
	}

	return builds, nil
}

func (d *Database) DeleteBuild(id string) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).Delete(&models.Build{}).Error; err != nil {
			return err
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
		Find(&builds).Error

	if err != nil {
		return nil, err
	}

	return builds, nil
}
