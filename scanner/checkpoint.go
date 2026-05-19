package scanner

import (
	"encoding/json"
	"net"
	"os"
	"time"
)

const (
	checkpointFile = "scan_checkpoint.json"
	saveInterval   = 100
)

type CheckpointPhase string

const (
	PhasePing  CheckpointPhase = "ping"
	PhaseSpeed CheckpointPhase = "speed"
	PhaseDone  CheckpointPhase = "done"
)

type cpPingResult struct {
	IP       string `json:"ip"`
	Sended   int    `json:"sended"`
	Received int    `json:"received"`
	DelayMs  int64  `json:"delay_ms"`
}

type Checkpoint struct {
	Mode          int             `json:"mode"`
	Workers       int             `json:"workers"`
	Phase         CheckpointPhase `json:"phase"`
	Completed     bool            `json:"completed"`
	ProgressIndex int             `json:"progress_index"`
	TotalIPs      int             `json:"total_ips"`
	AllIPs        []string        `json:"all_ips"`
	PingResults   []cpPingResult  `json:"ping_results"`
	SavedAt       string          `json:"saved_at"`
}

func NewCheckpoint(mode, workers int, ips []*net.IPAddr) *Checkpoint {
	allIPs := make([]string, len(ips))
	for i, ip := range ips {
		allIPs[i] = ip.String()
	}
	return &Checkpoint{
		Mode:     mode,
		Workers:  workers,
		Phase:    PhasePing,
		TotalIPs: len(ips),
		AllIPs:   allIPs,
	}
}

func (c *Checkpoint) SetPingResults(results []PingResult) {
	c.PingResults = make([]cpPingResult, len(results))
	for i, r := range results {
		c.PingResults[i] = cpPingResult{
			IP:       r.IP.String(),
			Sended:   r.Sended,
			Received: r.Received,
			DelayMs:  r.Delay.Milliseconds(),
		}
	}
}

func (c *Checkpoint) GetPingResults() []PingResult {
	results := make([]PingResult, 0, len(c.PingResults))
	for _, r := range c.PingResults {
		ipAddr, err := net.ResolveIPAddr("ip", r.IP)
		if err != nil {
			continue
		}
		results = append(results, PingResult{
			IP:       ipAddr,
			Sended:   r.Sended,
			Received: r.Received,
			Delay:    time.Duration(r.DelayMs) * time.Millisecond,
		})
	}
	return results
}

func (c *Checkpoint) GetRemainingIPs() []*net.IPAddr {
	start := c.ProgressIndex
	if start >= len(c.AllIPs) {
		return nil
	}
	ips := make([]*net.IPAddr, 0, len(c.AllIPs)-start)
	for _, ipStr := range c.AllIPs[start:] {
		ipAddr, err := net.ResolveIPAddr("ip", ipStr)
		if err != nil {
			continue
		}
		ips = append(ips, ipAddr)
	}
	return ips
}

func (c *Checkpoint) save() error {
	c.SavedAt = time.Now().Format("2006-01-02 15:04:05")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := checkpointFile + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, checkpointFile)
}

func (c *Checkpoint) Save() {
	c.save()
}

func (c *Checkpoint) MarkPingDone(allPingResults []PingResult) {
	c.Phase = PhaseSpeed
	c.ProgressIndex = c.TotalIPs
	c.SetPingResults(allPingResults)
	c.save()
}

func (c *Checkpoint) MarkCompleted() {
	c.Completed = true
	c.Phase = PhaseDone
	c.save()
}

func LoadCheckpoint() *Checkpoint {
	data, err := os.ReadFile(checkpointFile)
	if err != nil {
		return nil
	}
	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil
	}
	if cp.Completed || len(cp.AllIPs) == 0 {
		return nil
	}
	return &cp
}

func DeleteCheckpoint() {
	os.Remove(checkpointFile)
	os.Remove(checkpointFile + ".tmp")
}