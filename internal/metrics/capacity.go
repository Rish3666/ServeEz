package metrics

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/Rish3666/ServeEz/internal/api"
	"golang.org/x/sys/unix"
)

func DetectCapacity(root string) (api.Resources, error) {
	if root == "" {
		root = "/"
	}
	cpu := float64(runtime.NumCPU())
	if cpu < 1 {
		cpu = 1
	}
	mem, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return api.Resources{}, err
	}
	var st unix.Statfs_t
	if err := unix.Statfs(root, &st); err != nil {
		return api.Resources{}, err
	}
	if st.Blocks == 0 || st.Bsize == 0 {
		return api.Resources{}, errors.New("disk capacity unavailable")
	}
	disk := st.Blocks * uint64(st.Bsize)
	if disk == 0 {
		return api.Resources{}, fmt.Errorf("disk capacity unavailable")
	}
	return api.Resources{
		CPUCores:  cpu,
		MemBytes:  mem,
		DiskBytes: disk,
	}, nil
}
