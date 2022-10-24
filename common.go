package main

import "github.com/vvampirius/parcel-tracker/belpost"

func IsInSliceString(where []string, what string) bool {
	for _, value := range where {
		if value == what { return true }
	}
	return false
}


func BelpostSteps2TrackSteps(belpostApiResponse belpost.ApiResponse) []TrackStep {
	trackSteps := make([]TrackStep, 0)
	if !belpostApiResponse.IsFound() { return trackSteps }
	for _, data := range belpostApiResponse.Data {
		for _, belpostStep := range data.Steps {
			step := TrackStep{
				Provider: `Belpost`,
				Time: belpostStep.Time(),
				Event: belpostStep.Event,
				Place: belpostStep.Place,
			}
			if step.Event == `Принято в обработку в учреждении международного почтового обмена` { step.Important = true }
			if step.Event == `Попытка доставки` { step.Important = true }
			if step.Event == `Вручено` { step.Finish = true }
			trackSteps = append(trackSteps, step)
		}
	}
	if trackStepsCount := len(trackSteps); trackStepsCount > 0 {
		trackSteps[trackStepsCount - 1].Important = true
	}
	return trackSteps
}