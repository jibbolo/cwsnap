package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/russross/blackfriday"
)

var additionalFields = map[string]interface{}{
	"width":    600,
	"height":   300,
	"start":    "-PT12H",
	"end":      "P0D",
	"timezone": "+0200",
}

type imageGenerator struct {
	dashboards    map[string]*dashboardBody
	svc           *cloudwatch.Client
	dashboardList []string
}

func newImageGenerator() (*imageGenerator, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, errors.New("unable to load SDK config, " + err.Error())
	}

	// Set the AWS Region that the service clients should use
	cfg.Region = endpoints.UsWest2RegionID

	return &imageGenerator{
		dashboards: make(map[string]*dashboardBody),
		svc:        cloudwatch.New(cfg),
	}, nil
}

func (g *imageGenerator) refreshDashboardList() error {
	req := g.svc.ListDashboardsRequest(&cloudwatch.ListDashboardsInput{})
	resp, err := req.Send(context.Background())
	if err != nil {
		return fmt.Errorf("failed to describe dashboard: %v ", err)
	}
	g.dashboardList = make([]string, 0)
	for _, d := range resp.DashboardEntries {
		g.dashboardList = append(g.dashboardList, *d.DashboardName)
	}
	return nil
}

func (g *imageGenerator) refreshBody(dashboardName string) error {
	println("refreshing body for " + dashboardName)

	req := g.svc.GetDashboardRequest(&cloudwatch.GetDashboardInput{
		DashboardName: aws.String(dashboardName),
	})
	resp, err := req.Send(context.Background())
	if err != nil {
		return fmt.Errorf("failed to describe dashboard: %v ", err)
	}
	body, _ := g.dashboards[dashboardName]
	err = json.Unmarshal([]byte(*resp.DashboardBody), &body)
	if err != nil {
		return fmt.Errorf("failed to unmarshal: ", err)
	}
	g.dashboards[dashboardName] = body
	return nil
}

func (g *imageGenerator) renderWidget(dashboardName string, widgetIndex int) ([]byte, error) {

	if g.dashboards[dashboardName] == nil {
		if err := g.refreshBody(dashboardName); err != nil {
			return []byte{}, fmt.Errorf("renderWidget can't refresh body: %v ", err)
		}
	}

	widget := g.dashboards[dashboardName].Widgets[widgetIndex]
	for k, v := range additionalFields {
		widget.Properties[k] = v
	}
	props, err := json.Marshal(widget.Properties)
	req2 := g.svc.GetMetricWidgetImageRequest(&cloudwatch.GetMetricWidgetImageInput{
		MetricWidget: aws.String(string(props)),
	})

	resp2, err := req2.Send(context.Background())
	if err != nil {
		return []byte{}, fmt.Errorf("failed to describe widget: %v ", err)
	}
	return resp2.MetricWidgetImage, nil
}

type dashboardBody struct {
	Widgets []widget
}

type widget struct {
	Properties map[string]interface{}
}

func (w *widget) HasMarkdown() bool {
	_, ok := w.Properties["markdown"]
	return ok
}
func (w *widget) Markdown() string {
	if markdown, ok := w.Properties["markdown"]; ok {
		return string(blackfriday.Run([]byte(markdown.(string))))
	}
	return "Error: markdown unavailable"
}
