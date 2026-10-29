// Package learn holds the explanations of the learning mode: topics with a one-line hint and texts
// for three audiences (T-039). The texts are German and embedded from topics/*.json.
package learn

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"

	"galaxy/internal/i18n"
)

// Level is the audience of a text.
type Level int

const (
	School  Level = iota // pupils: vivid, no formulas
	Lay                  // interested laypeople: connections, orders of magnitude
	Physics              // physicists: equations, model assumptions, limits
	levels
)

// Key is the name of the level in the JSON files.
func (l Level) Key() string { return [...]string{"school", "lay", "physics"}[l] }

// Name is the label shown to the user.
func (l Level) Name() string {
	return [...]string{i18n.T("Schüler", "pupils"), i18n.T("Interessierte", "interested laypeople"), i18n.T("Physik", "physicists")}[l]
}

// Next and Prev cycle through the levels.
func (l Level) Next() Level { return (l + 1) % levels }
func (l Level) Prev() Level { return (l + levels - 1) % levels }

// Topic is one concept.
type Topic struct {
	ID     string            `json:"id"`
	Title  string            `json:"title"`
	Keys   string            `json:"keys"`  // related keys, informational
	Short  string            `json:"short"` // one-line hint
	Levels map[string]string `json:"levels"`
}

// Text returns the explanation for a level.
func (t *Topic) Text(l Level) string { return t.Levels[l.Key()] }

// Order is the order in which topics are listed.
var Order = []string{
	"gravity", "dark-matter", "black-holes", "hole-formation", "isco", "kerr", "accretion-disks", "speed-of-light", "doppler", "lensing",
	"retarded-gravity", "cosmic-web", "expansion", "spiral-arms", "tides", "gas", "cooling", "star-formation", "supernovae", "ionization", "dust", "energy", "units",
}

//go:embed topics/*.json
var topicFS embed.FS

var topics = mustLoad()

func mustLoad() map[string]*Topic {
	m, err := load()
	if err != nil {
		panic("learning topics: " + err.Error()) // embedded files, covered by tests
	}
	return m
}

func load() (map[string]*Topic, error) {
	names, err := fs.Glob(topicFS, "topics/*.json")
	if err != nil {
		return nil, err
	}
	m := map[string]*Topic{}
	for _, name := range names {
		data, err := topicFS.ReadFile(name)
		if err != nil {
			return nil, err
		}
		var t Topic
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		m[t.ID] = &t
	}
	return m, nil
}

// Get returns a topic by id, or nil.
func Get(id string) *Topic { return topics[id] }

// IDs returns all topic ids in display order (Order first, then any others sorted).
func IDs() []string {
	ids := slices.Clone(Order)
	var rest []string
	for id := range topics {
		if !slices.Contains(ids, id) {
			rest = append(rest, id)
		}
	}
	slices.Sort(rest)
	return append(ids, rest...)
}
