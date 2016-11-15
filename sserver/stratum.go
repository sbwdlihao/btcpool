package sserver

type StratumJob struct {
	JobId string
	PrevHash string
	Coinb1 string
	Coinb2 string
	MerkleBranch []string
	Version string
	Nbits string
	Ntime string
	CleanJobs bool
}
