# google-weather-cli

A minimal command-line weather tool powered by the [Google Weather API](https://developers.google.com/maps/documentation/weather). Zero dependencies beyond Go's standard library.

## Install

```sh
go install github.com/ptrotter1/google-weather-cli@latest
```

Or build from source:

```sh
git clone https://github.com/ptrotter1/google-weather-cli.git
cd google-weather-cli
go install .
```

## Setup

You need a Google Maps API key with the **Weather API** enabled.

1. Create or select a project in the [Google Cloud Console](https://console.cloud.google.com/)
2. Enable the [Weather API](https://console.cloud.google.com/apis/library/weather.googleapis.com)
3. Create an [API key](https://console.cloud.google.com/apis/credentials)
4. Export it:

```sh
export GOOGLE_MAPS_KEY="your-api-key"
```

## Usage

```
google-weather-cli <command> <location>
```

### Commands

| Command | Aliases | Description |
|---------|---------|-------------|
| `current` | `now` | Current weather conditions |
| `forecast` | `daily` | Multi-day daily forecast |
| `hourly` | | 24-hour hourly forecast |
| `version` | `-v`, `--version` | Print version |
| `help` | `-h`, `--help` | Print usage |

### Examples

```sh
# Current conditions
google-weather-cli current "New York"
google-weather-cli now Tokyo

# 5-day forecast
google-weather-cli forecast "San Francisco, CA"
google-weather-cli daily London

# Hourly forecast
google-weather-cli hourly "Chicago, IL"
google-weather-cli hourly 60601
```

### Sample output

```
  ☁️  Chicago, South Chicago Township, Cook County, Illinois, United States
  66°F (19°C) — Cloudy

  Feels like   66°F (19°C)
  Humidity     70%
  Wind         17 mph ↑ (gusts 25 mph)
  Dew point    56°F (13°C)
  UV index     0
  Visibility   10 mi
  Pressure     1008 hPa
  Cloud cover  91%
  Precip       10% (rain)
```

```
  📅  Chicago, South Chicago Township, Cook County, Illinois, United States — 5 day forecast

  2026-03-30  🌥️  Mostly cloudy
       High 71°F (21°C)  Low 48°F (9°C)
       Night: 🌦️ Light rain
       💧 45%  2.5mm  💨 14 mph  ⛈️  60%
       🌅 6:36 AM  🌇 7:13 PM
```

## How it works

- **Geocoding** — Location names are resolved to coordinates using [Nominatim](https://nominatim.openstreetmap.org/) (OpenStreetMap). No extra API keys or setup required.
- **Weather data** — Fetched from the [Google Weather API](https://developers.google.com/maps/documentation/weather) using your `GOOGLE_MAPS_KEY`.
- **Single binary** — One `main.go` file, no external dependencies. Built entirely with Go's standard library.

