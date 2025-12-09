package library

import (
	"testing"
)

func TestParseTVEpisodeWithTitle(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		dir         string
		wantTitle   string
		wantSeason  int
		wantEpisode int
		wantEpTitle string
	}{
		{
			name:        "The Rookie format with episode title and quality",
			filename:    "The Rookie - S01E02 - Crash Course WEBDL-1080p",
			dir:         "/media/tv",
			wantTitle:   "The Rookie",
			wantSeason:  1,
			wantEpisode: 2,
			wantEpTitle: "Crash Course",
		},
		{
			name:        "Dotted format with episode title",
			filename:    "Breaking.Bad.S01E02.Cat's.in.the.Bag.720p",
			dir:         "/media/tv",
			wantTitle:   "Breaking Bad",
			wantSeason:  1,
			wantEpisode: 2,
			wantEpTitle: "Cat's In The Bag",
		},
		{
			name:        "Simple format without episode title",
			filename:    "Show.Name.S01E02.mkv",
			dir:         "/media/tv",
			wantTitle:   "Show Name",
			wantSeason:  1,
			wantEpisode: 2,
			wantEpTitle: "",
		},
		{
			name:        "1x02 format with episode title",
			filename:    "The Office - 1x02 - Diversity Day",
			dir:         "/media/tv",
			wantTitle:   "The Office",
			wantSeason:  1,
			wantEpisode: 2,
			wantEpTitle: "Diversity Day",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTVEpisode(tt.filename, tt.dir)
			if result == nil {
				t.Fatal("parseTVEpisode returned nil")
			}

			if result.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", result.Title, tt.wantTitle)
			}
			if result.Season != tt.wantSeason {
				t.Errorf("Season = %d, want %d", result.Season, tt.wantSeason)
			}
			if result.Episode != tt.wantEpisode {
				t.Errorf("Episode = %d, want %d", result.Episode, tt.wantEpisode)
			}
			if result.EpisodeTitle != tt.wantEpTitle {
				t.Errorf("EpisodeTitle = %q, want %q", result.EpisodeTitle, tt.wantEpTitle)
			}
		})
	}
}
