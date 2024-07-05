package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	ipInfoAPIURL         = "https://ipinfo.io/"
	ipInfoAPIKey         = "b54ee3d6c3c552"           
	openWeatherMapAPIURL = "https://api.openweathermap.org/data/2.5/weather"
	openWeatherMapAPIKey = "a947cd07b70b649ccf74dc1b5df94bb1" 
)

type Location struct {
	IP       string `json:"ip"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Location string `json:"loc"`
}

type Weather struct {
	Main struct {
		Temp float64 `json:"temp"`
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"`
	} `json:"weather"`
}

func getIpAddr(req *http.Request) string {
	visitorIp := req.Header.Get("X-FORWARDED-FOR")
	if visitorIp != "" {
		// Return the first IP in the list
		return strings.Split(visitorIp, ",")[0]
	}
	return req.RemoteAddr
}

func getGeoLocation(ip string) (*Location, error) {
	url := fmt.Sprintf("%s%s?token=%s", ipInfoAPIURL, ip, ipInfoAPIKey)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get location: %s", resp.Status)
	}

	var location Location
	if err := json.NewDecoder(resp.Body).Decode(&location); err != nil {
		return nil, err
	}

	return &location, nil
}

func getWeather(city string) (*Weather, error) {
	url := fmt.Sprintf("%s?q=%s&appid=%s&units=metric", openWeatherMapAPIURL, url.QueryEscape(city), openWeatherMapAPIKey)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get weather: %s, response: %s", resp.Status, string(bodyBytes))
	}

	var weather Weather
	if err := json.NewDecoder(resp.Body).Decode(&weather); err != nil {
		return nil, err
	}

	return &weather, nil
}

func respHandler(resp http.ResponseWriter, req *http.Request) {
	clientIp := getIpAddr(req)
	if strings.Contains(clientIp, ":") {
		// If the IP address contains a port, remove the port
		clientIp = strings.Split(clientIp, ":")[0]
	}

	location, err := getGeoLocation(clientIp)
	if err != nil {
		http.Error(resp, fmt.Sprintf("Error getting location: %v", err), http.StatusInternalServerError)
		return
	}

	if location.City == "" {
		http.Error(resp, "Error: City not found in location data", http.StatusInternalServerError)
		return
	}

	weather, err := getWeather(location.City)
	if err != nil {
		http.Error(resp, fmt.Sprintf("Error getting weather: %v", err), http.StatusInternalServerError)
		return
	}

	// Split the "loc" field into latitude and longitude
	var latitude, longitude float64
	fmt.Sscanf(location.Location, "%f,%f", &latitude, &longitude)

	// Extract visitor name from query parameters
	query := req.URL.Query()
	visitor := query.Get("visitor")

	if visitor == "" {
		visitor = "Guest"
	}

	response := fmt.Sprintf(
		"Hello %s, the temperature is %.2f degrees Celsius in %s.\nIP: %s\nCity: %s\nRegion: %s\nCountry: %s\nLatitude: %f\nLongitude: %f\n",
		visitor, weather.Main.Temp, location.City, location.IP, location.City, location.Region, location.Country, latitude, longitude,
	)

	fmt.Fprint(resp, response)
	fmt.Println("New IP:", clientIp)
	fmt.Println(response)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", respHandler)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
