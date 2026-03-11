package microvm

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// VMPool maintains a pool of pre-warmed Firecracker VMs.
// On job arrival we grab a VM from the pool (no boot latency),
// execute the job, then discard the VM and replenish the pool.
type VMPool struct {
	mu          sync.Mutex
	available   []*VM
	kernelPath  string
	rootfsPath  string
	snapshotDir string
	size        int
}

// NewVMPool creates a warm pool of `size` VMs.
func NewVMPool(ctx context.Context, kernelPath, snapshotDir string, size int) (*VMPool, error) {
	p := &VMPool{
		kernelPath:  kernelPath,
		snapshotDir: snapshotDir,
		size:        size,
	}
	// Pre-warm.
	for i := 0; i < size; i++ {
		vm, err := p.bootVM(ctx, fmt.Sprintf("pool-%d", i))
		if err != nil {
			return nil, fmt.Errorf("warm vm %d: %w", i, err)
		}
		p.available = append(p.available, vm)
	}
	slog.Info("vm pool ready", "size", size)
	return p, nil
}

// Acquire returns a VM from the pool. Blocks until one is available.
func (p *VMPool) Acquire(ctx context.Context) (*VM, error) {
	for {
		p.mu.Lock()
		if len(p.available) > 0 {
			vm := p.available[0]
			p.available = p.available[1:]
			p.mu.Unlock()
			return vm, nil
		}
		p.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
}

// Release stops the used VM and adds a fresh one back to the pool.
func (p *VMPool) Release(ctx context.Context, vm *VM) {
	_ = vm.Stop()
	go func() {
		fresh, err := p.bootVM(ctx, vm.id)
		if err != nil {
			slog.Error("replenish vm pool", "err", err)
			return
		}
		p.mu.Lock()
		p.available = append(p.available, fresh)
		p.mu.Unlock()
	}()
}

func (p *VMPool) bootVM(ctx context.Context, id string) (*VM, error) {
	return NewVM(ctx, id, p.snapshotDir+"/state", p.kernelPath)
}
