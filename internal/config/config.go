// Package config loads config/leagues.yaml.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/lioilsources/BestBet/internal/hockey"
)

// File is the whole config.
type File struct {
	Seasons []int    `yaml:"seasons"`
	Leagues []League `yaml:"leagues"`
	// Aliases: source -> league slug -> alias -> canonical team name.
	Aliases map[string]map[string]map[string]string `yaml:"aliases"`
}

// League is one competition.
type League struct {
	Slug      string          `yaml:"slug"`
	Name      string          `yaml:"name"`
	Country   string          `yaml:"country"`
	Role      string          `yaml:"role"`
	TZ        string          `yaml:"tz"`
	OT        hockey.OTFormat `yaml:"ot"`
	APIHockey struct {
		ID      int      `yaml:"id"`
		Search  string   `yaml:"search"`
		Country string   `yaml:"country"`
		Alt     []string `yaml:"alt"`
	} `yaml:"apihockey"`
	OddsPapi struct {
		TournamentID int    `yaml:"tournament_id"`
		Search       string `yaml:"search"`
		Category     string `yaml:"category"`
	} `yaml:"oddspapi"`
	Swehockey []SwehockeySchedule `yaml:"swehockey"`
}

// SwehockeySchedule is one schedule page on stats.swehockey.se.
type SwehockeySchedule struct {
	Season     int    `yaml:"season"`
	ScheduleID int    `yaml:"schedule_id"`
	Stage      string `yaml:"stage"`
}

// Location returns the league time zone.
func (l League) Location() (*time.Location, error) {
	loc, err := time.LoadLocation(l.TZ)
	if err != nil {
		return nil, fmt.Errorf("league %s tz %q: %w", l.Slug, l.TZ, err)
	}
	return loc, nil
}

// Load reads and validates the file.
func Load(path string) (File, error) {
	var f File
	b, err := os.ReadFile(path)
	if err != nil {
		return f, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(b, &f); err != nil {
		return f, fmt.Errorf("parse config: %w", err)
	}
	seen := map[string]bool{}
	for _, l := range f.Leagues {
		if l.Slug == "" || seen[l.Slug] {
			return f, fmt.Errorf("league slug %q empty or duplicate", l.Slug)
		}
		seen[l.Slug] = true
		if _, err := l.Location(); err != nil {
			return f, err
		}
		switch l.Role {
		case "fringe", "semi_efficient", "control":
		default:
			return f, fmt.Errorf("league %s: role %q", l.Slug, l.Role)
		}
	}
	return f, nil
}

// League finds a league by slug.
func (f File) League(slug string) (League, bool) {
	for _, l := range f.Leagues {
		if l.Slug == slug {
			return l, true
		}
	}
	return League{}, false
}
