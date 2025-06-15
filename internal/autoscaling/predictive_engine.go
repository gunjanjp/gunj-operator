package autoscaling

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PredictiveScalingEngine implements predictive scaling algorithms
type PredictiveScalingEngine struct {
	modelType string
	data      []MetricDataPoint
	model     PredictiveModel
}

// NewPredictiveScalingEngine creates a new predictive scaling engine
func NewPredictiveScalingEngine(modelType string) *PredictiveScalingEngine {
	return &PredictiveScalingEngine{
		modelType: modelType,
		data:      []MetricDataPoint{},
	}
}

// Train trains the predictive model with historical data
func (e *PredictiveScalingEngine) Train(ctx context.Context, data []MetricDataPoint) error {
	log := log.FromContext(ctx)
	
	if len(data) < 10 {
		return fmt.Errorf("insufficient data points for training: need at least 10, got %d", len(data))
	}
	
	e.data = data
	
	switch e.modelType {
	case "linear":
		return e.trainLinearModel(ctx)
	case "exponential":
		return e.trainExponentialSmoothing(ctx)
	case "seasonal":
		return e.trainSeasonalModel(ctx)
	default:
		log.Info("Unknown model type, falling back to linear", "modelType", e.modelType)
		return e.trainLinearModel(ctx)
	}
}

// Predict makes predictions for the given time horizon
func (e *PredictiveScalingEngine) Predict(ctx context.Context, horizon time.Duration) ([]MetricDataPoint, error) {
	if len(e.data) == 0 {
		return nil, fmt.Errorf("model not trained: no data available")
	}
	
	switch e.modelType {
	case "linear":
		return e.predictLinear(ctx, horizon)
	case "exponential":
		return e.predictExponentialSmoothing(ctx, horizon)
	case "seasonal":
		return e.predictSeasonal(ctx, horizon)
	default:
		return e.predictLinear(ctx, horizon)
	}
}

// GetAccuracy returns the model's accuracy
func (e *PredictiveScalingEngine) GetAccuracy() float64 {
	return e.model.Accuracy
}

// GetModelType returns the model type
func (e *PredictiveScalingEngine) GetModelType() string {
	return e.modelType
}

// trainLinearModel trains a simple linear regression model
func (e *PredictiveScalingEngine) trainLinearModel(ctx context.Context) error {
	log := log.FromContext(ctx)
	
	// Sort data by timestamp
	sort.Slice(e.data, func(i, j int) bool {
		return e.data[i].Timestamp.Before(e.data[j].Timestamp)
	})
	
	// Convert timestamps to numeric values (seconds since first observation)
	baseTime := e.data[0].Timestamp
	x := make([]float64, len(e.data))
	y := make([]float64, len(e.data))
	
	for i, point := range e.data {
		x[i] = point.Timestamp.Sub(baseTime).Seconds()
		y[i] = point.Value
	}
	
	// Calculate linear regression coefficients
	n := float64(len(x))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	
	for i := range x {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
	}
	
	// Calculate slope (m) and intercept (b) for y = mx + b
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / n
	
	// Calculate R-squared for accuracy
	meanY := sumY / n
	ssTotal, ssResidual := 0.0, 0.0
	
	for i := range x {
		predicted := slope*x[i] + intercept
		ssTotal += math.Pow(y[i]-meanY, 2)
		ssResidual += math.Pow(y[i]-predicted, 2)
	}
	
	rSquared := 1 - (ssResidual / ssTotal)
	
	e.model = PredictiveModel{
		ModelType: "linear",
		Accuracy:  rSquared,
		LastUpdated: time.Now(),
		TrainingData: e.data,
	}
	
	// Store model parameters in labels
	e.model.TrainingData[0].Labels = map[string]string{
		"slope":     fmt.Sprintf("%f", slope),
		"intercept": fmt.Sprintf("%f", intercept),
		"baseTime":  baseTime.Format(time.RFC3339),
	}
	
	log.Info("Linear model trained", "slope", slope, "intercept", intercept, "accuracy", rSquared)
	
	return nil
}

// predictLinear makes predictions using the linear model
func (e *PredictiveScalingEngine) predictLinear(ctx context.Context, horizon time.Duration) ([]MetricDataPoint, error) {
	if len(e.model.TrainingData) == 0 {
		return nil, fmt.Errorf("model not trained")
	}
	
	// Extract model parameters
	params := e.model.TrainingData[0].Labels
	slope, _ := parseFloat(params["slope"])
	intercept, _ := parseFloat(params["intercept"])
	baseTime, _ := time.Parse(time.RFC3339, params["baseTime"])
	
	// Generate predictions
	predictions := []MetricDataPoint{}
	lastDataPoint := e.data[len(e.data)-1]
	predictionStart := lastDataPoint.Timestamp
	
	// Create predictions at 5-minute intervals
	interval := 5 * time.Minute
	steps := int(horizon / interval)
	
	for i := 0; i < steps; i++ {
		timestamp := predictionStart.Add(time.Duration(i) * interval)
		timeSinceBase := timestamp.Sub(baseTime).Seconds()
		value := slope*timeSinceBase + intercept
		
		// Ensure non-negative predictions
		if value < 0 {
			value = 0
		}
		
		predictions = append(predictions, MetricDataPoint{
			Timestamp: timestamp,
			Value:     value,
			Labels: map[string]string{
				"type": "prediction",
			},
		})
	}
	
	return predictions, nil
}

// trainExponentialSmoothing trains an exponential smoothing model
func (e *PredictiveScalingEngine) trainExponentialSmoothing(ctx context.Context) error {
	log := log.FromContext(ctx)
	
	// Sort data by timestamp
	sort.Slice(e.data, func(i, j int) bool {
		return e.data[i].Timestamp.Before(e.data[j].Timestamp)
	})
	
	// Simple exponential smoothing with optimized alpha
	bestAlpha := e.optimizeAlpha()
	
	// Calculate smoothed values
	smoothed := make([]float64, len(e.data))
	smoothed[0] = e.data[0].Value
	
	for i := 1; i < len(e.data); i++ {
		smoothed[i] = bestAlpha*e.data[i].Value + (1-bestAlpha)*smoothed[i-1]
	}
	
	// Calculate accuracy (MAPE - Mean Absolute Percentage Error)
	mape := 0.0
	for i := 1; i < len(e.data); i++ {
		if e.data[i].Value != 0 {
			mape += math.Abs((e.data[i].Value-smoothed[i-1])/e.data[i].Value) * 100
		}
	}
	mape /= float64(len(e.data) - 1)
	accuracy := math.Max(0, 1-mape/100)
	
	e.model = PredictiveModel{
		ModelType:    "exponential",
		Accuracy:     accuracy,
		LastUpdated:  time.Now(),
		TrainingData: e.data,
	}
	
	// Store model parameters
	e.model.TrainingData[0].Labels = map[string]string{
		"alpha":        fmt.Sprintf("%f", bestAlpha),
		"lastSmoothed": fmt.Sprintf("%f", smoothed[len(smoothed)-1]),
	}
	
	log.Info("Exponential smoothing model trained", "alpha", bestAlpha, "accuracy", accuracy)
	
	return nil
}

// optimizeAlpha finds the optimal smoothing parameter
func (e *PredictiveScalingEngine) optimizeAlpha() float64 {
	bestAlpha := 0.3
	bestError := math.MaxFloat64
	
	// Try different alpha values
	for alpha := 0.1; alpha <= 0.9; alpha += 0.1 {
		totalError := 0.0
		smoothed := e.data[0].Value
		
		for i := 1; i < len(e.data); i++ {
			predicted := smoothed
			actual := e.data[i].Value
			error := math.Abs(actual - predicted)
			totalError += error
			
			smoothed = alpha*actual + (1-alpha)*smoothed
		}
		
		if totalError < bestError {
			bestError = totalError
			bestAlpha = alpha
		}
	}
	
	return bestAlpha
}

// predictExponentialSmoothing makes predictions using exponential smoothing
func (e *PredictiveScalingEngine) predictExponentialSmoothing(ctx context.Context, horizon time.Duration) ([]MetricDataPoint, error) {
	if len(e.model.TrainingData) == 0 {
		return nil, fmt.Errorf("model not trained")
	}
	
	// Extract model parameters
	params := e.model.TrainingData[0].Labels
	alpha, _ := parseFloat(params["alpha"])
	lastSmoothed, _ := parseFloat(params["lastSmoothed"])
	
	// For simple exponential smoothing, forecast is constant
	predictions := []MetricDataPoint{}
	lastDataPoint := e.data[len(e.data)-1]
	predictionStart := lastDataPoint.Timestamp
	
	interval := 5 * time.Minute
	steps := int(horizon / interval)
	
	// Add trend detection
	trend := e.detectTrend()
	
	for i := 0; i < steps; i++ {
		timestamp := predictionStart.Add(time.Duration(i) * interval)
		
		// Adjust prediction based on trend
		value := lastSmoothed + trend*float64(i)
		
		// Ensure non-negative predictions
		if value < 0 {
			value = 0
		}
		
		predictions = append(predictions, MetricDataPoint{
			Timestamp: timestamp,
			Value:     value,
			Labels: map[string]string{
				"type": "prediction",
			},
		})
	}
	
	return predictions, nil
}

// trainSeasonalModel trains a model that accounts for seasonal patterns
func (e *PredictiveScalingEngine) trainSeasonalModel(ctx context.Context) error {
	log := log.FromContext(ctx)
	
	// Detect seasonality period (e.g., daily, weekly)
	period := e.detectSeasonality()
	
	if period == 0 {
		log.Info("No seasonality detected, falling back to linear model")
		return e.trainLinearModel(ctx)
	}
	
	// Calculate seasonal indices
	seasonalIndices := e.calculateSeasonalIndices(period)
	
	// Store model parameters
	e.model = PredictiveModel{
		ModelType:        "seasonal",
		Accuracy:         0.85, // Placeholder - would calculate actual accuracy
		LastUpdated:      time.Now(),
		TrainingData:     e.data,
		PredictionWindow: time.Duration(period) * time.Hour,
	}
	
	// Store seasonal indices in labels
	for i, index := range seasonalIndices {
		label := fmt.Sprintf("seasonal_%d", i)
		e.model.TrainingData[0].Labels[label] = fmt.Sprintf("%f", index)
	}
	
	log.Info("Seasonal model trained", "period", period, "indices", len(seasonalIndices))
	
	return nil
}

// detectSeasonality detects the seasonal period in hours
func (e *PredictiveScalingEngine) detectSeasonality() int {
	// Simple detection: check for daily (24h) and weekly (168h) patterns
	// In production, use more sophisticated methods like FFT or autocorrelation
	
	if len(e.data) < 48 { // Need at least 2 days of hourly data
		return 0
	}
	
	// Check for daily pattern
	dailyCorrelation := e.calculateAutocorrelation(24)
	weeklyCorrelation := e.calculateAutocorrelation(168)
	
	if dailyCorrelation > 0.7 {
		return 24
	} else if weeklyCorrelation > 0.7 {
		return 168
	}
	
	return 0
}

// calculateAutocorrelation calculates autocorrelation for a given lag
func (e *PredictiveScalingEngine) calculateAutocorrelation(lag int) float64 {
	if len(e.data) < lag*2 {
		return 0
	}
	
	// Calculate mean
	sum := 0.0
	for _, point := range e.data {
		sum += point.Value
	}
	mean := sum / float64(len(e.data))
	
	// Calculate autocorrelation
	numerator := 0.0
	denominator := 0.0
	
	for i := lag; i < len(e.data); i++ {
		numerator += (e.data[i].Value - mean) * (e.data[i-lag].Value - mean)
	}
	
	for _, point := range e.data {
		denominator += math.Pow(point.Value-mean, 2)
	}
	
	if denominator == 0 {
		return 0
	}
	
	return numerator / denominator
}

// calculateSeasonalIndices calculates seasonal adjustment factors
func (e *PredictiveScalingEngine) calculateSeasonalIndices(period int) []float64 {
	indices := make([]float64, period)
	counts := make([]int, period)
	
	// Initialize
	for i := range indices {
		indices[i] = 0
		counts[i] = 0
	}
	
	// Calculate average for each seasonal position
	for i, point := range e.data {
		position := i % period
		indices[position] += point.Value
		counts[position]++
	}
	
	// Calculate averages
	overallSum := 0.0
	validCount := 0
	
	for i := range indices {
		if counts[i] > 0 {
			indices[i] /= float64(counts[i])
			overallSum += indices[i]
			validCount++
		}
	}
	
	// Normalize indices
	if validCount > 0 {
		overallAvg := overallSum / float64(validCount)
		for i := range indices {
			if counts[i] > 0 {
				indices[i] /= overallAvg
			} else {
				indices[i] = 1.0
			}
		}
	}
	
	return indices
}

// predictSeasonal makes predictions using the seasonal model
func (e *PredictiveScalingEngine) predictSeasonal(ctx context.Context, horizon time.Duration) ([]MetricDataPoint, error) {
	// For simplicity, use the last observed pattern
	return e.predictLinear(ctx, horizon)
}

// detectTrend detects the trend in the data
func (e *PredictiveScalingEngine) detectTrend() float64 {
	if len(e.data) < 2 {
		return 0
	}
	
	// Simple trend: difference between last and first value divided by time
	firstValue := e.data[0].Value
	lastValue := e.data[len(e.data)-1].Value
	timeDiff := e.data[len(e.data)-1].Timestamp.Sub(e.data[0].Timestamp).Hours()
	
	if timeDiff == 0 {
		return 0
	}
	
	return (lastValue - firstValue) / timeDiff
}

// parseFloat safely parses a string to float64
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// MakePrediction makes a scaling prediction based on the trained model
func (e *PredictiveScalingEngine) MakePrediction(ctx context.Context, component v1beta1.ComponentType, currentReplicas int32, horizon time.Duration) (*ScalingDecision, error) {
	predictions, err := e.Predict(ctx, horizon)
	if err != nil {
		return nil, err
	}
	
	if len(predictions) == 0 {
		return nil, fmt.Errorf("no predictions generated")
	}
	
	// Find the peak predicted value
	maxValue := 0.0
	for _, pred := range predictions {
		if pred.Value > maxValue {
			maxValue = pred.Value
		}
	}
	
	// Calculate required replicas based on prediction
	// Assume each replica can handle 100 units of load
	replicasPerUnit := 100.0
	targetReplicas := int32(math.Ceil(maxValue / replicasPerUnit))
	
	// Apply bounds
	if targetReplicas < 1 {
		targetReplicas = 1
	}
	if targetReplicas > 10 {
		targetReplicas = 10
	}
	
	return &ScalingDecision{
		Type:            PredictiveScaling,
		Component:       component,
		CurrentReplicas: currentReplicas,
		TargetReplicas:  targetReplicas,
		Reason:          fmt.Sprintf("Predicted load of %.2f units in next %v", maxValue, horizon),
		Timestamp:       time.Now(),
	}, nil
}
