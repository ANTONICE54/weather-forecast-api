package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"weather-forecast/internal/domain/models"
	"weather-forecast/internal/infrastructure/apperrors"
	"weather-forecast/internal/infrastructure/logger"
)

const LOCATION_NOT_FOUND = 1006

type (
	getWeatherResponse struct {
		Current struct {
			TempC     float64 `json:"temp_c"`
			Condition struct {
				Text string `json:"text"`
			} `json:"condition"`
			Humidity int `json:"humidity"`
		} `json:"current"`
	}

	getWeatherErrorResponse struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	WeatherProvider struct {
		apiURL string
		apiKey string
		client *http.Client
		logger logger.Logger
	}
)

func NewWeatherProvider(apiURL, apiKey string, httpClient *http.Client, logger logger.Logger) *WeatherProvider {
	return &WeatherProvider{
		apiURL: apiURL,
		apiKey: apiKey,
		client: httpClient,
		logger: logger,
	}
}

func (p *WeatherProvider) GetWeatherByCity(ctx context.Context, city string) (*models.Weather, error) {
	url := p.apiURL + fmt.Sprintf("?key=%s&q=%s", p.apiKey, city)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		p.logger.Warnf("Failed to create get weather request: %s", err.Error())
		return nil, apperrors.GetWeatherError
	}

	resp, err := p.client.Do(req)
	if err != nil {
		p.logger.Warnf("Failed make get weather request: %s", err.Error())
		return nil, apperrors.GetWeatherError
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		p.logger.Warnf("Failed to read response body: %s", err.Error())
		return nil, apperrors.GetWeatherError
	}

	if resp.StatusCode != http.StatusOK {
		var errResponse getWeatherErrorResponse

		if err := json.Unmarshal(body, &errResponse); err != nil {
			p.logger.Warnf("Failed to unmarshal response body: %s", err.Error())
			return nil, apperrors.GetWeatherError
		}

		if errResponse.Error.Code == LOCATION_NOT_FOUND {
			p.logger.Warnf("City not found: %s", city)
			return nil, apperrors.CityNotFoundError
		} else {
			p.logger.Warnf("Error from weather api: %s", errResponse.Error.Message)
			return nil, apperrors.GetWeatherError
		}

	}

	var weather getWeatherResponse

	if err := json.Unmarshal(body, &weather); err != nil {
		p.logger.Warnf("Failed to unmarshal response body: %s", err.Error())
		return nil, apperrors.GetWeatherError
	}

	result := models.Weather{
		Temperature: weather.Current.TempC,
		Humidity:    weather.Current.Humidity,
		Description: weather.Current.Condition.Text,
	}

	return &result, nil
}
