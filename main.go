package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

const (
	nominatimBase = "https://nominatim.openstreetmap.org/search"
	weatherBase   = "https://weather.googleapis.com/v1"
)

// --- Weather types (matching actual API responses) ---

type temperature struct {
	Degrees float64 `json:"degrees"`
	Unit    string  `json:"unit"`
}

type weatherCondition struct {
	Description struct {
		Text string `json:"text"`
	} `json:"description"`
	Type string `json:"type"`
}

type precipitation struct {
	Probability struct {
		Percent int    `json:"percent"`
		Type    string `json:"type"`
	} `json:"probability"`
	Qpf struct {
		Quantity float64 `json:"quantity"`
		Unit     string  `json:"unit"`
	} `json:"qpf"`
	SnowQpf struct {
		Quantity float64 `json:"quantity"`
		Unit     string  `json:"unit"`
	} `json:"snowQpf"`
}

type wind struct {
	Direction struct {
		Degrees  int    `json:"degrees"`
		Cardinal string `json:"cardinal"`
	} `json:"direction"`
	Speed struct {
		Value float64 `json:"value"`
		Unit  string  `json:"unit"`
	} `json:"speed"`
	Gust struct {
		Value float64 `json:"value"`
		Unit  string  `json:"unit"`
	} `json:"gust"`
}

type visibility struct {
	Distance float64 `json:"distance"`
	Unit     string  `json:"unit"`
}

type airPressure struct {
	MeanSeaLevelMillibars float64 `json:"meanSeaLevelMillibars"`
}

type interval struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

type currentConditionsResponse struct {
	CurrentTime          string           `json:"currentTime"`
	TimeZone             struct{ ID string `json:"id"` } `json:"timeZone"`
	IsDaytime            bool             `json:"isDaytime"`
	WeatherCondition     weatherCondition `json:"weatherCondition"`
	Temperature          temperature      `json:"temperature"`
	FeelsLikeTemperature temperature      `json:"feelsLikeTemperature"`
	DewPoint             temperature      `json:"dewPoint"`
	HeatIndex            temperature      `json:"heatIndex"`
	WindChill            temperature      `json:"windChill"`
	RelativeHumidity     int              `json:"relativeHumidity"`
	UVIndex              int              `json:"uvIndex"`
	Precipitation        precipitation    `json:"precipitation"`
	ThunderstormProb     int              `json:"thunderstormProbability"`
	AirPressure          airPressure      `json:"airPressure"`
	Wind                 wind             `json:"wind"`
	Visibility           visibility       `json:"visibility"`
	CloudCover           int              `json:"cloudCover"`
}

type forecastResponse struct {
	ForecastDays []forecastDay `json:"forecastDays"`
	TimeZone     struct{ ID string `json:"id"` } `json:"timeZone"`
}

type dayPartForecast struct {
	WeatherCondition     weatherCondition `json:"weatherCondition"`
	RelativeHumidity     int              `json:"relativeHumidity"`
	UVIndex              int              `json:"uvIndex"`
	Precipitation        precipitation    `json:"precipitation"`
	ThunderstormProb     int              `json:"thunderstormProbability"`
	Wind                 wind             `json:"wind"`
	CloudCover           int              `json:"cloudCover"`
}

type forecastDay struct {
	Interval         interval `json:"interval"`
	DisplayDate      struct {
		Year  int `json:"year"`
		Month int `json:"month"`
		Day   int `json:"day"`
	} `json:"displayDate"`
	DaytimeForecast   dayPartForecast `json:"daytimeForecast"`
	NighttimeForecast dayPartForecast `json:"nighttimeForecast"`
	MaxTemperature    temperature     `json:"maxTemperature"`
	MinTemperature    temperature     `json:"minTemperature"`
	SunEvents         struct {
		SunriseTime string `json:"sunriseTime"`
		SunsetTime  string `json:"sunsetTime"`
	} `json:"sunEvents"`
	MoonEvents struct {
		MoonPhase string `json:"moonPhase"`
	} `json:"moonEvents"`
}

type hourlyForecastResponse struct {
	ForecastHours []forecastHour `json:"forecastHours"`
}

type forecastHour struct {
	Interval             interval         `json:"interval"`
	DisplayDateTime      struct {
		Hours int `json:"hours"`
	} `json:"displayDateTime"`
	WeatherCondition     weatherCondition `json:"weatherCondition"`
	Temperature          temperature      `json:"temperature"`
	FeelsLikeTemperature temperature      `json:"feelsLikeTemperature"`
	DewPoint             temperature      `json:"dewPoint"`
	Precipitation        precipitation    `json:"precipitation"`
	ThunderstormProb     int              `json:"thunderstormProbability"`
	AirPressure          airPressure      `json:"airPressure"`
	Wind                 wind             `json:"wind"`
	Visibility           visibility       `json:"visibility"`
	IsDaytime            bool             `json:"isDaytime"`
	RelativeHumidity     int              `json:"relativeHumidity"`
	UVIndex              int              `json:"uvIndex"`
	CloudCover           int              `json:"cloudCover"`
}

// --- Client ---

type client struct {
	apiKey     string
	httpClient *http.Client
}

func newClient() (*client, error) {
	key := os.Getenv("GOOGLE_MAPS_KEY")
	if key == "" {
		return nil, fmt.Errorf("GOOGLE_MAPS_KEY environment variable is not set")
	}
	return &client{apiKey: key, httpClient: &http.Client{}}, nil
}

func (c *client) geocode(query string) (float64, float64, string, error) {
	u := fmt.Sprintf("%s?q=%s&format=json&limit=1", nominatimBase, url.QueryEscape(query))
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return 0, 0, "", err
	}
	req.Header.Set("User-Agent", "google-weather-cli")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, 0, "", fmt.Errorf("geocode request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, "", fmt.Errorf("geocode returned status %d", resp.StatusCode)
	}

	var results []struct {
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return 0, 0, "", fmt.Errorf("decoding geocode response: %w", err)
	}

	if len(results) == 0 {
		return 0, 0, "", fmt.Errorf("could not find location: %s", query)
	}

	r := results[0]
	var lat, lng float64
	fmt.Sscanf(r.Lat, "%f", &lat)
	fmt.Sscanf(r.Lon, "%f", &lng)
	return lat, lng, r.DisplayName, nil
}

func (c *client) weatherGet(endpoint string, lat, lng float64, extraParams map[string]string, out any) error {
	u := fmt.Sprintf("%s/%s?key=%s&location.latitude=%f&location.longitude=%f",
		weatherBase, endpoint, c.apiKey, lat, lng)
	for k, v := range extraParams {
		u += "&" + k + "=" + v
	}

	resp, err := c.httpClient.Get(u)
	if err != nil {
		return fmt.Errorf("weather request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("weather API returned status %d: %s", resp.StatusCode, string(b))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding weather response: %w", err)
	}
	return nil
}

// --- Formatting helpers ---

func celsiusToFahrenheit(c float64) float64 {
	return c*9/5 + 32
}

func kphToMph(kph float64) float64 {
	return kph * 0.621371
}

func kmToMi(km float64) float64 {
	return km * 0.621371
}

func windArrow(deg int) string {
	arrows := []string{"↓", "↙", "←", "↖", "↑", "↗", "→", "↘"}
	return arrows[((deg+22)%360)/45]
}

func weatherIcon(condType string, isDaytime bool) string {
	switch {
	case strings.Contains(condType, "CLEAR"):
		if isDaytime {
			return "☀️"
		}
		return "🌙"
	case strings.Contains(condType, "PARTLY"):
		if isDaytime {
			return "⛅"
		}
		return "☁️"
	case strings.Contains(condType, "MOSTLY_CLOUDY"):
		return "🌥️"
	case strings.Contains(condType, "CLOUDY"):
		return "☁️"
	case strings.Contains(condType, "FOG") || strings.Contains(condType, "HAZE"):
		return "🌫️"
	case strings.Contains(condType, "THUNDER"):
		return "⛈️"
	case strings.Contains(condType, "HEAVY_RAIN"):
		return "🌧️"
	case strings.Contains(condType, "RAIN"):
		return "🌦️"
	case strings.Contains(condType, "SNOW"):
		return "🌨️"
	case strings.Contains(condType, "SLEET") || strings.Contains(condType, "ICE"):
		return "🧊"
	case strings.Contains(condType, "WIND"):
		return "💨"
	default:
		return "🌡️"
	}
}

func formatTemp(degrees float64) string {
	return fmt.Sprintf("%.0f°F (%.0f°C)", celsiusToFahrenheit(degrees), degrees)
}

func formatWind(w wind) string {
	return fmt.Sprintf("%.0f mph %s", kphToMph(w.Speed.Value), windArrow(w.Direction.Degrees))
}

func formatWindWithGust(w wind) string {
	s := formatWind(w)
	if w.Gust.Value > w.Speed.Value {
		s += fmt.Sprintf(" (gusts %.0f mph)", kphToMph(w.Gust.Value))
	}
	return s
}

func shortTime(iso string) string {
	if idx := strings.Index(iso, "T"); idx >= 0 {
		t := iso[idx+1:]
		if end := strings.Index(t, "."); end >= 0 {
			t = t[:end]
		}
		if end := strings.Index(t, "Z"); end >= 0 {
			t = t[:end]
		}
		if len(t) >= 5 {
			return t[:5]
		}
		return t
	}
	return iso
}

func localTime(iso string, tzID string) string {
	t, err := time.Parse(time.RFC3339Nano, iso)
	if err != nil {
		return shortTime(iso) + " UTC"
	}
	loc, err := time.LoadLocation(tzID)
	if err != nil {
		return t.Format("3:04 PM") + " UTC"
	}
	return t.In(loc).Format("3:04 PM")
}

func formatHour(h int) string {
	switch {
	case h == 0:
		return "12 AM"
	case h < 12:
		return fmt.Sprintf("%d AM", h)
	case h == 12:
		return "12 PM"
	default:
		return fmt.Sprintf("%d PM", h-12)
	}
}

// --- Commands ---

func cmdCurrent(c *client, location string) error {
	lat, lng, addr, err := c.geocode(location)
	if err != nil {
		return err
	}

	var weather currentConditionsResponse
	if err := c.weatherGet("currentConditions:lookup", lat, lng, nil, &weather); err != nil {
		return err
	}

	icon := weatherIcon(weather.WeatherCondition.Type, weather.IsDaytime)
	desc := weather.WeatherCondition.Description.Text

	fmt.Printf("\n  %s  %s\n", icon, addr)
	fmt.Printf("  %s — %s\n\n", formatTemp(weather.Temperature.Degrees), desc)
	fmt.Printf("  Feels like   %s\n", formatTemp(weather.FeelsLikeTemperature.Degrees))
	fmt.Printf("  Humidity     %d%%\n", weather.RelativeHumidity)
	fmt.Printf("  Wind         %s\n", formatWindWithGust(weather.Wind))
	fmt.Printf("  Dew point    %s\n", formatTemp(weather.DewPoint.Degrees))
	fmt.Printf("  UV index     %d\n", weather.UVIndex)
	fmt.Printf("  Visibility   %.0f mi\n", kmToMi(weather.Visibility.Distance))
	fmt.Printf("  Pressure     %.0f hPa\n", weather.AirPressure.MeanSeaLevelMillibars)
	fmt.Printf("  Cloud cover  %d%%\n", weather.CloudCover)
	if weather.Precipitation.Probability.Percent > 0 {
		fmt.Printf("  Precip       %d%% (%s)\n", weather.Precipitation.Probability.Percent, strings.ToLower(weather.Precipitation.Probability.Type))
	}
	if weather.ThunderstormProb > 0 {
		fmt.Printf("  Storms       %d%%\n", weather.ThunderstormProb)
	}
	fmt.Println()
	return nil
}

func cmdForecast(c *client, location string) error {
	lat, lng, addr, err := c.geocode(location)
	if err != nil {
		return err
	}

	var forecast forecastResponse
	if err := c.weatherGet("forecast/days:lookup", lat, lng, nil, &forecast); err != nil {
		return err
	}

	fmt.Printf("\n  📅  %s — %d day forecast\n\n", addr, len(forecast.ForecastDays))

	for _, day := range forecast.ForecastDays {
		date := fmt.Sprintf("%d-%02d-%02d", day.DisplayDate.Year, day.DisplayDate.Month, day.DisplayDate.Day)
		dayIcon := weatherIcon(day.DaytimeForecast.WeatherCondition.Type, true)
		dayDesc := day.DaytimeForecast.WeatherCondition.Description.Text
		nightDesc := day.NighttimeForecast.WeatherCondition.Description.Text

		fmt.Printf("  %s  %s  %s\n", date, dayIcon, dayDesc)
		fmt.Printf("       High %s  Low %s\n", formatTemp(day.MaxTemperature.Degrees), formatTemp(day.MinTemperature.Degrees))

		if nightDesc != dayDesc {
			nightIcon := weatherIcon(day.NighttimeForecast.WeatherCondition.Type, false)
			fmt.Printf("       Night: %s %s\n", nightIcon, nightDesc)
		}

		details := []string{}
		dayPrecip := day.DaytimeForecast.Precipitation.Probability.Percent
		nightPrecip := day.NighttimeForecast.Precipitation.Probability.Percent
		if dayPrecip > 0 || nightPrecip > 0 {
			maxPrecip := dayPrecip
			if nightPrecip > maxPrecip {
				maxPrecip = nightPrecip
			}
			details = append(details, fmt.Sprintf("💧 %d%%", maxPrecip))
		}
		totalQpf := day.DaytimeForecast.Precipitation.Qpf.Quantity + day.NighttimeForecast.Precipitation.Qpf.Quantity
		if totalQpf > 0 {
			details = append(details, fmt.Sprintf("%.1fmm", totalQpf))
		}
		details = append(details, fmt.Sprintf("💨 %.0f mph", kphToMph(day.DaytimeForecast.Wind.Speed.Value)))
		dayStorm := day.DaytimeForecast.ThunderstormProb
		nightStorm := day.NighttimeForecast.ThunderstormProb
		if dayStorm > 0 || nightStorm > 0 {
			maxStorm := dayStorm
			if nightStorm > maxStorm {
				maxStorm = nightStorm
			}
			details = append(details, fmt.Sprintf("⛈️  %d%%", maxStorm))
		}
		fmt.Printf("       %s\n", strings.Join(details, "  "))

		if day.SunEvents.SunriseTime != "" && day.SunEvents.SunsetTime != "" {
			fmt.Printf("       🌅 %s  🌇 %s\n",
				localTime(day.SunEvents.SunriseTime, forecast.TimeZone.ID),
				localTime(day.SunEvents.SunsetTime, forecast.TimeZone.ID))
		}
		fmt.Println()
	}
	return nil
}

func cmdHourly(c *client, location string) error {
	lat, lng, addr, err := c.geocode(location)
	if err != nil {
		return err
	}

	var forecast hourlyForecastResponse
	if err := c.weatherGet("forecast/hours:lookup", lat, lng, map[string]string{"hours": "24"}, &forecast); err != nil {
		return err
	}

	fmt.Printf("\n  🕐  %s — 24 hour forecast\n\n", addr)

	for _, hour := range forecast.ForecastHours {
		t := formatHour(hour.DisplayDateTime.Hours)
		icon := weatherIcon(hour.WeatherCondition.Type, hour.IsDaytime)
		precip := ""
		if hour.Precipitation.Probability.Percent > 0 {
			precip = fmt.Sprintf("  💧 %d%%", hour.Precipitation.Probability.Percent)
		}
		fmt.Printf("  %-5s  %s  %-16s  %s%s\n",
			t, icon, formatTemp(hour.Temperature.Degrees),
			formatWind(hour.Wind),
			precip,
		)
	}
	fmt.Println()
	return nil
}

func cmdVersion() {
	version := "dev"
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && len(s.Value) >= 7 {
				version = s.Value[:7]
				break
			}
		}
	}
	fmt.Printf("google-weather-cli %s\n", version)
}

func cmdHelp() {
	fmt.Print(`Usage: google-weather-cli <command> <location>

Commands:
  current <location>   Show current weather conditions
  forecast <location>  Show multi-day forecast
  hourly <location>    Show 24-hour forecast
  version              Show version
  help                 Show this help

Environment:
  GOOGLE_MAPS_KEY      Google Maps API key (required)

Examples:
  google-weather-cli current "New York"
  google-weather-cli forecast "San Francisco, CA"
  google-weather-cli hourly Tokyo
`)
}

func run() error {
	if len(os.Args) < 2 {
		cmdHelp()
		return nil
	}

	cmd := os.Args[1]

	switch cmd {
	case "version", "--version", "-v":
		cmdVersion()
		return nil
	case "help", "--help", "-h":
		cmdHelp()
		return nil
	}

	if len(os.Args) < 3 {
		return fmt.Errorf("missing location argument\nUsage: google-weather-cli %s <location>", cmd)
	}
	location := strings.Join(os.Args[2:], " ")

	c, err := newClient()
	if err != nil {
		return err
	}

	switch cmd {
	case "current", "now":
		return cmdCurrent(c, location)
	case "forecast", "daily":
		return cmdForecast(c, location)
	case "hourly":
		return cmdHourly(c, location)
	default:
		return fmt.Errorf("unknown command: %s\nRun 'google-weather-cli help' for usage", cmd)
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
