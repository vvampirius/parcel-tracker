module github.com/vvampirius/parcel-tracker

go 1.17

replace github.com/vvampirius/parcel-tracker/config => ./config

replace github.com/vvampirius/parcel-tracker/belpost => ./belpost

require (
	github.com/vvampirius/mygolibs/telegram v0.0.0-20221107075418-a12919befab0
	gopkg.in/yaml.v3 v3.0.1
)
