// Package layers contains the layer structure of CYPHONIC Packet.
package layers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
)

// AcquisitionRequest executes the General node Information Acquisition process.
func AcquisitionRequest(httpClient *http.Client, ep string) ([]AcquisitionResponse, error) {
	var ares []AcquisitionResponse

	res, err := httpClient.Get(ep)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP request end point: %w", err)
	}
	defer func() error {
		if err = res.Body.Close(); err != nil {
			return fmt.Errorf("failed to close http conncetion: %w", err)
		}
		return nil
	}()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("error: status code: %v", res.StatusCode)
	}

	logger.Debug(fmt.Sprintf("Send Acquisition Request to %s", ep))

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP response body: %w", err)
	}

	if err := json.Unmarshal(body, &ares); err != nil {
		return nil, fmt.Errorf("failed to json unmarshal Acquisition Response: %w", err)
	}

	logger.Debug(fmt.Sprintf("Receive Acquisition Response code: %d", res.StatusCode))

	return ares, nil
}
