package main

import (
	"gopkg.in/yaml.v3"
	"os"
	"path"
)

type Tracks struct {
	dirPath string
}

func (tracks *Tracks) Load(id string) (Track, error) {
	track := Track{}
	f, err := os.Open(path.Join(tracks.dirPath, id))
	if err != nil {
		if !os.IsNotExist(err) { ErrorLog.Println(id, err.Error()) }
		return track, err
	}
	defer f.Close()
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&track); err != nil {
		ErrorLog.Println(id, err.Error())
		return track, err
	}
	return track, nil
}

func (tracks *Tracks) Save(track Track) error {
	f, err := os.OpenFile(path.Join(tracks.dirPath, track.Id), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		ErrorLog.Println(err.Error())
		return err
	}
	defer f.Close()
	encoder := yaml.NewEncoder(f)
	if err := encoder.Encode(track); err != nil {
		ErrorLog.Println(err.Error())
		return err
	}
	return nil
}

func (tracks *Tracks) Get(id string) Track {
	if track, err := tracks.Load(id); err == nil { return track }
	return NewTrack(id)
}

func NewTracks(dirPath string) (*Tracks, error) {
	if err := os.MkdirAll(dirPath, 0744); err != nil {
		ErrorLog.Println(dirPath, err.Error())
		return nil, err
	}
	tracks := Tracks{
		dirPath: dirPath,
	}
	return &tracks, nil
}