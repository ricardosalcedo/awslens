package aws

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// Sparkline returns a tiny 12-char sparkline for a metric over the last 24h.
func (c *Client) Sparkline(ctx context.Context, namespace, metric, dimName, dimValue string) string {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	svc := cloudwatch.NewFromConfig(c.Config)
	end := time.Now()
	start := end.Add(-24 * time.Hour)

	out, err := svc.GetMetricStatistics(ctx, &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metric),
		Dimensions: []cwtypes.Dimension{{Name: aws.String(dimName), Value: aws.String(dimValue)}},
		StartTime:  aws.Time(start),
		EndTime:    aws.Time(end),
		Period:     aws.Int32(7200), // 2h buckets = 12 points
		Statistics: []cwtypes.Statistic{cwtypes.StatisticAverage},
	})
	if err != nil || len(out.Datapoints) == 0 {
		return ""
	}

	// sort by timestamp
	points := make([]float64, 12)
	for _, dp := range out.Datapoints {
		idx := int(dp.Timestamp.Sub(start).Hours() / 2)
		if idx >= 0 && idx < 12 {
			points[idx] = aws.ToFloat64(dp.Average)
		}
	}
	return renderSparkline(points)
}

func renderSparkline(values []float64) string {
	bars := []rune("▁▂▃▄▅▆▇█")
	max := 0.0
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	if max == 0 {
		return strings.Repeat("▁", len(values))
	}
	result := make([]rune, len(values))
	for i, v := range values {
		idx := int(v / max * 7)
		if idx > 7 {
			idx = 7
		}
		result[i] = bars[idx]
	}
	return string(result)
}
