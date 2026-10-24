package app

import (
	"slices"

	"galaxy/internal/learn"
)

// Learning mode (T-039): a panel with the scene introduction and explanations of the concepts.
// Triggering a feature shows a one-line hint and focuses its topic.

// IntroTopic is the pseudo topic for the scene introduction (first entry of the list).
const IntroTopic = ""

// topicList is the scene introduction, the scene's topics, then all others.
func topicList(sceneTopics []string) []string {
	list := append([]string{IntroTopic}, sceneTopics...)
	for _, id := range learn.IDs() {
		if !slices.Contains(list, id) {
			list = append(list, id)
		}
	}
	return list
}

// hint shows the one-line hint of a topic and makes it the one the panel opens.
func (c *Controller) hint(id string) {
	t := learn.Get(id)
	if t == nil {
		return
	}
	c.focus(id)
	c.Notify(t.Short + "   (e: mehr)")
}

func (c *Controller) focus(id string) {
	if k := slices.Index(c.Topics, id); k >= 0 {
		c.TopicIdx = k
	}
}

// OpenExplanation opens the panel for an audience ("school", "lay", "physics") and topic ("" for
// the scene introduction); used by the -explain flag and for screenshots.
func (c *Controller) OpenExplanation(level, topic string) {
	for l := learn.School; l <= learn.Physics; l++ {
		if l.Key() == level {
			c.Level = l
		}
	}
	c.focus(topic)
	c.Explain = true
}

// Explanation returns title and text of the focused entry.
func (c *Controller) Explanation() (title, body string) {
	id := c.Topics[c.TopicIdx]
	if id == IntroTopic {
		return c.Scene.Name, c.Scene.Intro[c.Level.Key()]
	}
	t := learn.Get(id)
	body = t.Text(c.Level)
	if t.Keys != "" {
		body += "\n\nTasten: " + t.Keys
	}
	return t.Title, body
}
