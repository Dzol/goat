package main

import "net/http"
import "encoding/json"
import "strings"
import "log"
import "time"
import "fmt"

type weatherProvider interface {
    temperature(city string) (float64, error)
}

type openWeatherMap struct {
	key string
}

func (w openWeatherMap) temperature(city string) (float64, error) {
	resp, err := http.Get("http://api.openweathermap.org/data/2.5/weather?APPID=" + w.key + "&q=" + city)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	var d struct {
		Main struct {
			Kelvin float64 `json:"temp"`
		} `json:"main"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return 0, err
	}

	log.Printf("openWeatherMap: %s @ %.2f", city, d.Main.Kelvin)
	return d.Main.Kelvin, nil
}

type weatherUnderground struct {
	key string
}

func (w weatherUnderground) temperature(city string) (float64, error) {
    resp, err := http.Get("http://api.wunderground.com/api/" + w.key + "/conditions/q/" + city + ".json")
    if err != nil {
        return 0, err
    }

    defer resp.Body.Close()

    var d struct {
        Observation struct {
            Celsius float64 `json:"temp_c"`
        } `json:"current_observation"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return 0, err
    }

    kelvin := d.Observation.Celsius + 273.15
    log.Printf("weatherUnderground: %s @ %.2f", city, kelvin)
    return kelvin, nil
}

var owm = openWeatherMap{key: ""}
var wu = weatherUnderground{key: ""}

func main() {
	http.HandleFunc("/greet", greet)
	http.HandleFunc("/weather/", func(w http.ResponseWriter, r *http.Request) {
		begin := time.Now()
		city := strings.SplitN(r.URL.Path, "/", 3)[2]

		temp, err := temperature(city, owm, wu)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"city": city,
			"temp": temp,
			"took": time.Since(begin).String(),
		})

	})
	http.ListenAndServe(":8080", nil)
}

func greet(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hey Joe!"))
}

func temperature(city string, providers ...weatherProvider) (float64, error) {
	sum := 0.0
	c := make(chan float64, 2)

	for _, provider := range providers {
		go func(p weatherProvider) {
			k, err := p.temperature(city)
			if err != nil {
				fmt.Println(err)
			}
			c <- k
		}(provider)
	}

	for i := 0; i < len(providers); i++ {
		k := <-c
		sum += k
	}

	return sum / float64(len(providers)), nil
}
