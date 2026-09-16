package main

import (
	"context"
	"errors"
	"testing"

	"github.com/carloska24/nexus-core-lab/internal/device"
	"github.com/carloska24/nexus-core-lab/internal/network"
	"github.com/carloska24/nexus-core-lab/internal/session"
	"github.com/carloska24/nexus-core-lab/internal/subscriber"
)

func TestDeviceCheckerAdapterRequiresCurrentActiveSubscriber(t *testing.T) {
	ctx := context.Background()
	subscriberService := subscriber.NewService(subscriber.NewMemoryRepository())
	deviceService := device.NewService(device.NewMemoryRepository(), &subscriberCheckerAdapter{subService: subscriberService})

	sub, err := subscriberService.Provision(ctx, subscriber.ProvisionRequest{
		IMSI: "724000000000101", MSISDN: "+5519999990101",
	})
	if err != nil {
		t.Fatalf("provision failed: %v", err)
	}
	if _, err := subscriberService.Activate(ctx, sub.ID); err != nil {
		t.Fatalf("activate failed: %v", err)
	}
	dev, err := deviceService.Register(ctx, device.RegisterRequest{
		SubscriberID: sub.ID, IMEI: "860010000000101", Technology: device.Tech5G,
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	adapter := &deviceCheckerAdapter{deviceService: deviceService, subService: subscriberService}
	if _, err := adapter.CheckDeviceAttachable(ctx, dev.ID); err != nil {
		t.Fatalf("ACTIVE subscriber should be attachable: %v", err)
	}
	if _, err := subscriberService.Suspend(ctx, sub.ID, "gate-11-test"); err != nil {
		t.Fatalf("suspend failed: %v", err)
	}
	if _, err := adapter.CheckDeviceAttachable(ctx, dev.ID); !errors.Is(err, session.ErrSubscriberNotActive) {
		t.Fatalf("SUSPENDED subscriber should be rejected, got %v", err)
	}
	if _, err := subscriberService.Activate(ctx, sub.ID); err != nil {
		t.Fatalf("reactivate failed: %v", err)
	}
	if _, err := adapter.CheckDeviceAttachable(ctx, dev.ID); err != nil {
		t.Fatalf("reactivated subscriber should be attachable: %v", err)
	}
	if _, err := subscriberService.Deactivate(ctx, sub.ID, "gate-11-test"); err != nil {
		t.Fatalf("deactivate failed: %v", err)
	}
	if _, err := adapter.CheckDeviceAttachable(ctx, dev.ID); !errors.Is(err, session.ErrSubscriberNotActive) {
		t.Fatalf("DEACTIVATED subscriber should be rejected, got %v", err)
	}
}

func TestTechnologyRemainsSemanticMetadata(t *testing.T) {
	ctx := context.Background()
	subscriberService := subscriber.NewService(subscriber.NewMemoryRepository())
	deviceService := device.NewService(device.NewMemoryRepository(), &subscriberCheckerAdapter{subService: subscriberService})
	adapter := &deviceCheckerAdapter{deviceService: deviceService, subService: subscriberService}
	sessionService := session.NewService(session.NewMemoryRepository(), adapter, network.NewIPPool())

	cases := []struct {
		imsi, msisdn, imei string
		technology         device.Technology
		cellID             string
	}{
		{"724000000000201", "+5519999990201", "860010000000201", device.Tech5G, "CELL-SP-001"},
		{"724000000000202", "+5519999990202", "860010000000202", device.TechLTE, "CELL-SP-002"},
	}

	for _, tc := range cases {
		sub, err := subscriberService.Provision(ctx, subscriber.ProvisionRequest{IMSI: tc.imsi, MSISDN: tc.msisdn})
		if err != nil {
			t.Fatalf("provision failed: %v", err)
		}
		if _, err := subscriberService.Activate(ctx, sub.ID); err != nil {
			t.Fatalf("activate failed: %v", err)
		}
		dev, err := deviceService.Register(ctx, device.RegisterRequest{SubscriberID: sub.ID, IMEI: tc.imei, Technology: tc.technology})
		if err != nil {
			t.Fatalf("register failed: %v", err)
		}
		if _, err := sessionService.Attach(ctx, session.AttachRequest{DeviceID: dev.ID, CellID: tc.cellID}); err != nil {
			t.Fatalf("technology metadata must not block attach (%s -> %s): %v", tc.technology, tc.cellID, err)
		}
	}
}

func TestSuspensionPreservesConnectedSessionAndRejectsReattach(t *testing.T) {
	ctx := context.Background()
	subscriberService := subscriber.NewService(subscriber.NewMemoryRepository())
	deviceService := device.NewService(device.NewMemoryRepository(), &subscriberCheckerAdapter{subService: subscriberService})
	sessionService := session.NewService(
		session.NewMemoryRepository(),
		&deviceCheckerAdapter{deviceService: deviceService, subService: subscriberService},
		network.NewIPPool(),
	)

	sub, err := subscriberService.Provision(ctx, subscriber.ProvisionRequest{
		IMSI: "724000000000301", MSISDN: "+5519999990301",
	})
	if err != nil {
		t.Fatalf("provision failed: %v", err)
	}
	if _, err := subscriberService.Activate(ctx, sub.ID); err != nil {
		t.Fatalf("activate failed: %v", err)
	}
	dev, err := deviceService.Register(ctx, device.RegisterRequest{
		SubscriberID: sub.ID, IMEI: "860010000000301", Technology: device.Tech5G,
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	original, err := sessionService.Attach(ctx, session.AttachRequest{DeviceID: dev.ID, CellID: "CELL-SP-001"})
	if err != nil {
		t.Fatalf("initial attach failed: %v", err)
	}

	if _, err := subscriberService.Suspend(ctx, sub.ID, "gate-11-test"); err != nil {
		t.Fatalf("suspend failed: %v", err)
	}
	stillConnected, err := sessionService.GetActiveByDevice(ctx, dev.ID)
	if err != nil || stillConnected.ID != original.ID || stillConnected.IPAddress != original.IPAddress {
		t.Fatalf("suspension changed the connected session: session=%+v err=%v", stillConnected, err)
	}
	if _, err := sessionService.Attach(ctx, session.AttachRequest{DeviceID: dev.ID, CellID: "CELL-SP-002"}); !errors.Is(err, session.ErrSubscriberNotActive) {
		t.Fatalf("re-attach after suspension should be rejected, got %v", err)
	}
	afterRejectedAttach, err := sessionService.GetActiveByDevice(ctx, dev.ID)
	if err != nil || afterRejectedAttach.ID != original.ID || afterRejectedAttach.IPAddress != original.IPAddress {
		t.Fatalf("rejected re-attach changed the original session: session=%+v err=%v", afterRejectedAttach, err)
	}
}
