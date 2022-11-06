package main

import (
	"fmt"
	"github.com/vvampirius/parcel-tracker/belpost"
	"io"
)

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

// WriteSteps writes report to io.Writer and returns important flag
func WriteSteps(w io.Writer, steps []TrackStep) (bool, error) {
	important := false
	for _, step := range steps {
		if _, err := fmt.Fprintf(w, "⏱%s 🏤%s: %s\n", step.Time.Format(`02.01 15:04`), step.Place, step.Event); // TODO: correct to GMT+3 ?
		err != nil {
			ErrorLog.Println(err.Error())
			return important, err
		}
		if step.Important { important = true }
	}
	return important, nil
}