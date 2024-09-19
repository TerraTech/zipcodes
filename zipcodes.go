// zipcodes is a package that uses the GeoNames Postal Code dataset from http://www.geonames.org
// in order to perform zipcode lookup operations
package zipcodes

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

const (
	earthRadiusKm = 6371
	earthRadiusMi = 3958
)

var (
	ErrMultipleLatLon  = errors.New("zipcode has multiple lat/lon coordinates")
	ErrZipcodeNotFound = errors.New("zipcode not found")
)

// ZipCodeLocation struct represents each line of the dataset
type ZipCodeLocation struct {
	ZipCode   string
	PlaceName string
	AdminName string
	Lat       float64
	Lon       float64
	StateCode string
}

// ZipCodeLocations slice represents a zipcode with multiple lat/lon coordinates
type ZipCodeLocations []ZipCodeLocation

// Zipcodes contains the whole list of structs representing the zipcode dataset
type Zipcodes struct {
	DatasetList map[string]ZipCodeLocations
}

// New loads the dataset and returns a struct that contains the dataset as a map interface
func New(datasetPath string) (Zipcodes, error) {
	return LoadDataset(datasetPath)
}

// NewByCountry loads the dataset, filtered by country, and
// returns a struct that contains the dataset as a map interface
func NewByCountry(datasetPath, country string) (Zipcodes, error) {
	return LoadDatasetByCountry(datasetPath, country)
}

// Lookup looks for a zipcode inside the map interface
func (zc Zipcodes) Lookup(zipCode string) (ZipCodeLocations, error) {
	zipcodes, exists := zc.DatasetList[zipCode]
	if !exists {
		return nil, ErrZipcodeNotFound
	} else if len(zipcodes) > 1 {
		return zipcodes, ErrMultipleLatLon
	} else {
		return zipcodes, nil
	}
}

// DistanceInKm returns the line of sight distance between two zipcode locations in Kilometers
func (zc Zipcodes) DistanceInKm(zipcodeLocationA ZipCodeLocation, zipcodeLocationB ZipCodeLocation) float64 {
	return zc.CalculateDistance(zipcodeLocationA, zipcodeLocationB, earthRadiusKm)
}

// DistanceInMiles returns the line of sight distance between two zipcode locations in Miles
func (zc Zipcodes) DistanceInMiles(zipcodeLocationA, zipcodeLocationB ZipCodeLocation) float64 {
	return zc.CalculateDistance(zipcodeLocationA, zipcodeLocationB, earthRadiusMi)
}

// CalculateDistance returns the line of sight distance between two zipcode locations in Kilometers
func (zc Zipcodes) CalculateDistance(zipcodeLocationA, zipcodeLocationB ZipCodeLocation, radius float64) float64 {
	return DistanceBetweenPoints(zipcodeLocationA.Lat, zipcodeLocationA.Lon, zipcodeLocationB.Lat, zipcodeLocationB.Lon, radius)
}

// DistanceInKmToZipcode calculates the distance between a zipcode and a give lat/lon in Kilometers
func (zc Zipcodes) DistanceInKmToZipCode(zipcodeLocation ZipCodeLocation, latitude, longitude float64) float64 {
	return DistanceBetweenPoints(zipcodeLocation.Lat, zipcodeLocation.Lon, latitude, longitude, earthRadiusKm)
}

// DistanceInMilToZipcode calculates the distance between a zipcode and a give lat/lon in Miles
func (zc Zipcodes) DistanceInMilToZipCode(zipcodeLocation ZipCodeLocation, latitude, longitude float64) float64 {
	return DistanceBetweenPoints(zipcodeLocation.Lat, zipcodeLocation.Lon, latitude, longitude, earthRadiusMi)
}

// GetZipcodesWithinKmRadius get all zipcodes within the radius of this zipcode
func (zc Zipcodes) GetZipcodesWithinKmRadius(zipcodeLocation ZipCodeLocation, radius float64) []string {
	return zc.FindZipcodesWithinRadius(zipcodeLocation, radius, earthRadiusKm)
}

// GetZipcodesWithinMlRadius get all zipcodes within the radius of this zipcode
func (zc Zipcodes) GetZipcodesWithinMlRadius(zipcodeLocation ZipCodeLocation, radius float64) []string {
	return zc.FindZipcodesWithinRadius(zipcodeLocation, radius, earthRadiusMi)
}

// FindZipcodesWithinRadius finds zipcodes within a given radius
func (zc Zipcodes) FindZipcodesWithinRadius(zipcodeLocation ZipCodeLocation, maxRadius, earthRadius float64) []string {
	zipcodeList := []string{}
	for _, elm := range zc.DatasetList {
		for _, zcls := range elm {
			if zcls.ZipCode != zipcodeLocation.ZipCode {
				distance := DistanceBetweenPoints(zipcodeLocation.Lat, zipcodeLocation.Lon, zcls.Lat, zcls.Lon, earthRadius)
				if distance < maxRadius {
					zipcodeList = append(zipcodeList, zcls.ZipCode)
				}
			}
		}
	}

	return zipcodeList
}

// IsMulti returns if there are multiple lat/lon coordinates for a zipcode
func (zcl ZipCodeLocations) IsMulti() bool {
	return len(zcl) > 1
}

// DistanceBetweenPoints returns the distance between two lat/lon
// points using the Haversine distance formula.
func DistanceBetweenPoints(latitude1, longitude1, latitude2, longitude2, radius float64) float64 {
	lat1 := degreesToRadians(latitude1)
	lon1 := degreesToRadians(longitude1)
	lat2 := degreesToRadians(latitude2)
	lon2 := degreesToRadians(longitude2)
	diffLat := lat2 - lat1
	diffLon := lon2 - lon1

	a := hsin(diffLat) + math.Cos(lat1)*math.Cos(lat2)*hsin(diffLon)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := c * radius

	return math.Round(distance*100) / 100
}

// LoadDataset reads and loads the dataset into a map interface
func LoadDataset(datasetPath string) (Zipcodes, error) {
	return loadDataset(datasetPath, "")
}

// LoadDatasetByCountry reads and loads the dataset into a map interface filtered by ISO Country Code
func LoadDatasetByCountry(datasetPath, country string) (Zipcodes, error) {
	return loadDataset(datasetPath, country)
}

// IsMulti returns if there are multiple lat/lon coordinates for a zipcode
func IsMulti(zipcodeLocations ZipCodeLocations) bool {
	return zipcodeLocations.IsMulti()
}

// degreesToRadians converts degrees to radians
func degreesToRadians(d float64) float64 {
	return d * math.Pi / 180
}

func hsin(t float64) float64 {
	return math.Pow(math.Sin(t/2), 2)
}

// loadDataset is a consilidated function handling LoadDataset() and LoadDatasetByCountry()
func loadDataset(datasetPath, country string) (Zipcodes, error) {
	wantCountry := country != ""
	inCountry := false

	if wantCountry && len(country) != 2 {
		return Zipcodes{}, fmt.Errorf("country must be a 2 character ISO Country Code")
	}

	country = strings.ToUpper(country)

	file, err := os.Open(datasetPath)
	if err != nil {
		log.Fatal(err)
		return Zipcodes{}, fmt.Errorf("zipcodes: error while opening file %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	zipcodeMap := Zipcodes{DatasetList: make(map[string]ZipCodeLocations)}
	for scanner.Scan() {
		splittedLine := strings.Split(scanner.Text(), "\t")
		if len(splittedLine) != 12 {
			return Zipcodes{}, fmt.Errorf("zipcodes: file line does not have 12 fields")
		}

		if !wantCountry || splittedLine[0] == country {
			inCountry = true

			lat, errLat := strconv.ParseFloat(splittedLine[9], 64)
			if errLat != nil {
				return Zipcodes{}, fmt.Errorf("zipcodes: error while converting %s to Latitude", splittedLine[9])
			}

			lon, errLon := strconv.ParseFloat(splittedLine[10], 64)
			if errLon != nil {
				return Zipcodes{}, fmt.Errorf("zipcodes: error while converting %s to Longitude", splittedLine[10])
			}

			zipcodeMap.DatasetList[splittedLine[1]] =
				append(zipcodeMap.DatasetList[splittedLine[1]], ZipCodeLocation{
					ZipCode:   splittedLine[1],
					PlaceName: splittedLine[2],
					AdminName: splittedLine[3],
					Lat:       lat,
					Lon:       lon,
					StateCode: splittedLine[4],
				})
		} else if inCountry && splittedLine[0] != country {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return Zipcodes{}, fmt.Errorf("zipcodes: error while opening file %v", err)
	}

	return zipcodeMap, nil
}
