package cockroachdb



// JobResources represents the used cpu and memory resources for a job and the measured maximum once it's done
type JobResources struct {
	CPURequest     float64
	CPULimit       float64
	CPUMaxUsage    float64
	MemoryRequest  float64
	MemoryLimit    float64
	MemoryMaxUsage float64
}
