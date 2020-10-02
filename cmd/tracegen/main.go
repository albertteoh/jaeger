// Copyright (c) 2018 The Jaeger Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"github.com/jaegertracing/jaeger/cmd/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"time"

	"github.com/opentracing/opentracing-go"
	jaegerConfig "github.com/uber/jaeger-client-go/config"
	jaegerZap "github.com/uber/jaeger-client-go/log/zap"
	"github.com/uber/jaeger-lib/metrics/prometheus"
	"go.uber.org/zap"

	"github.com/jaegertracing/jaeger/internal/tracegen"
)

var logger, _ = zap.NewDevelopment()

func main() {
	fs := flag.CommandLine
	cfg := new(tracegen.Config)
	cfg.Flags(fs)
	flag.Parse()
	svc := flags.NewService(cfg.MetricsPort)
	svc.NoStorage = true

	v := viper.New()
	svc.AddFlags(fs)

	// Starts HTTP server listening on "/metrics" endpoint
	var command = &cobra.Command{}
	command.Flags().AddGoFlagSet(fs)
	v.BindPFlags(command.Flags())
	if err := svc.Start(v); err != nil {
		panic("failed to start service!")
	}

	metricsFactory := prometheus.New()
	traceCfg := &jaegerConfig.Configuration{
		ServiceName: cfg.Service,
		Sampler: &jaegerConfig.SamplerConfig{
			Type:                    "remote",
			SamplingRefreshInterval: time.Millisecond,
		},
		RPCMetrics: true,
	}
	traceCfg, err := traceCfg.FromEnv()
	if err != nil {
		logger.Fatal("failed to read tracer configuration", zap.Error(err))
	}

	tracer, tCloser, err := traceCfg.NewTracer(
		jaegerConfig.Metrics(metricsFactory),
		jaegerConfig.Logger(jaegerZap.NewLogger(logger)),
	)
	if err != nil {
		logger.Fatal("failed to create tracer", zap.Error(err))
	}
	defer tCloser.Close()

	opentracing.InitGlobalTracer(tracer)
	logger.Info("Initialized global tracer")

	tracegen.Run(cfg, logger)

	logger.Info("Waiting 1.5sec for metrics to flush")
	time.Sleep(3 * time.Second / 2)

}
