package mapillary

const mapillaryBaseUrl = "https://a.mapillary.com/v3/"

// zoom level at which the bbox are aligned (using OSM tile boundaries)
const gridZoomLevel = 15

// when retrieving data within a bounding box aligned to tiles, add this much border
// or overlap. 1 = 9 times the area of the bbox, so 0.05 = 5% border around tile
const tileBuffer = 0.05

// how many image details to fetch from Mapillary's private API per request
const imageDetailsChunkSize = 100

// how many meters of distance between two photos are allowed, before the
// Mapillary Viewer will not transition anymore.
const maxTransitionDistance = 25

type Config struct {
	FilterNewer string
	FilterUsers string
	APIKey      string
}
