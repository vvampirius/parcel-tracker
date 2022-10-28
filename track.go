package main

import "time"

type TrackStep struct {
	Time time.Time
	Event string
	Place string
	Provider string
	Important bool
	Finish bool
}

type Track struct {
	Id string
	Steps []TrackStep
}

func (track *Track) IsFinished() bool {
	for _, step := range track.Steps {
		if step.Finish { return true }
	}
	return false
}

// GetLastStepsTime returns last Time per provider
func (track *Track) GetLastStepsTime() map[string]time.Time {
	lasts := make(map[string]time.Time)
	for _, step := range track.Steps {
		last, found := lasts[step.Provider]
		if !found {
			lasts[step.Provider] = step.Time
			continue
		}
		if last.Before(step.Time) { lasts[step.Provider] = step.Time }
	}
	return lasts
}

// AddSteps returns added (new) steps
func (track *Track) AddSteps(steps []TrackStep) []TrackStep {
	lasts := track.GetLastStepsTime()
	addedSteps := make([]TrackStep, 0)
	for _, step := range steps {
		last, _ := lasts[step.Provider]
		if step.Time.After(last) {
			track.Steps = append(track.Steps, step)
			addedSteps = append(addedSteps, step)
		}
	}
	return addedSteps
}


func NewTrack(id string) Track {
	track := Track{
		Id: id,
		Steps: make([]TrackStep, 0),
	}
	return track
}
