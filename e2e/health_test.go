// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

var _ = Describe("Health Service E2E", func() {
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%s", "0.0.0.0", "31234"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	Expect(err).NotTo(HaveOccurred())

	client := healthpb.NewHealthClient(conn)

	Context("grpc.health.v1.Health", func() {
		It("should report SERVING for the whole server", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{})
			Expect(err).NotTo(HaveOccurred(), "health Check should not fail")
			Expect(resp.GetStatus()).To(Equal(healthpb.HealthCheckResponse_SERVING), "server should report SERVING")
		})

		It("should return NOT_FOUND for an unknown service name", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			_, err := client.Check(ctx, &healthpb.HealthCheckRequest{Service: "does.not.Exist"})
			Expect(err).To(HaveOccurred(), "unknown service should not be registered with the health service")
		})
	})
})
