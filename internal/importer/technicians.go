package importer

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/datsun80zx/sta.git/internal/db"
	"github.com/datsun80zx/sta.git/internal/parser"
)

// ImportTechnicians extracts technicians from jobs and creates relationships
func (i *Importer) ImportTechnicians(ctx context.Context, tx *sql.Tx, jobs []parser.JobRow, batchID int64) (int, error) {
	txQueries := db.New(tx)

	// Track unique technicians we've seen
	techCache := make(map[string]int64) // name -> id

	for _, job := range jobs {
		var completionDate *time.Time
		if job.JobCompletionDate != nil {
			completionDate = job.JobCompletionDate
		}

		// Process Sold By technician
		if job.SoldBy != nil && *job.SoldBy != "" {
			techID, err := i.upsertTechnician(ctx, txQueries, *job.SoldBy, completionDate, techCache)
			if err != nil {
				return 0, fmt.Errorf("failed to upsert sold_by technician: %w", err)
			}
			err = txQueries.CreateJobTechnician(ctx, db.CreateJobTechnicianParams{
				JobID:        job.JobID,
				TechnicianID: techID,
				Role:         "sold_by",
			})
			if err != nil {
				return 0, fmt.Errorf("failed to create job_technician (sold_by): %w", err)
			}
		}

		// Process Primary Technician
		if job.PrimaryTechnician != nil && *job.PrimaryTechnician != "" {
			techID, err := i.upsertTechnician(ctx, txQueries, *job.PrimaryTechnician, completionDate, techCache)
			if err != nil {
				return 0, fmt.Errorf("failed to upsert primary technician: %w", err)
			}
			err = txQueries.CreateJobTechnician(ctx, db.CreateJobTechnicianParams{
				JobID:        job.JobID,
				TechnicianID: techID,
				Role:         "primary",
			})
			if err != nil {
				return 0, fmt.Errorf("failed to create job_technician (primary): %w", err)
			}
		}

		// Process Assigned Technicians (can be comma-separated list)
		if job.AssignedTechnicians != nil && *job.AssignedTechnicians != "" {
			techNames := splitTechnicianNames(*job.AssignedTechnicians)
			for _, techName := range techNames {
				techID, err := i.upsertTechnician(ctx, txQueries, techName, completionDate, techCache)
				if err != nil {
					return 0, fmt.Errorf("failed to upsert assigned technician: %w", err)
				}
				err = txQueries.CreateJobTechnician(ctx, db.CreateJobTechnicianParams{
					JobID:        job.JobID,
					TechnicianID: techID,
					Role:         "assigned",
				})
				if err != nil {
					return 0, fmt.Errorf("failed to create job_technician (assigned): %w", err)
				}
			}
		}
	}

	return len(techCache), nil
}

// upsertTechnician creates or updates a technician and returns their ID
func (i *Importer) upsertTechnician(ctx context.Context, q *db.Queries, name string, jobDate *time.Time, cache map[string]int64) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("technician name cannot be empty")
	}

	// Check cache first
	if id, ok := cache[name]; ok {
		return id, nil
	}

	// Convert time.Time to sql.NullTime for the query
	var firstSeen, lastSeen sql.NullTime
	if jobDate != nil {
		firstSeen = sql.NullTime{Time: *jobDate, Valid: true}
		lastSeen = sql.NullTime{Time: *jobDate, Valid: true}
	}

	tech, err := q.UpsertTechnician(ctx, db.UpsertTechnicianParams{
		Name:          name,
		FirstSeenDate: firstSeen,
		LastSeenDate:  lastSeen,
	})
	if err != nil {
		return 0, err
	}

	cache[name] = tech.ID
	return tech.ID, nil
}

// splitTechnicianNames handles the comma-separated list of technician names
func splitTechnicianNames(names string) []string {
	var result []string
	parts := strings.Split(names, ",")
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name != "" {
			result = append(result, name)
		}
	}
	return result
}
