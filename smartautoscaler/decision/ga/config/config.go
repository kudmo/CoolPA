package config

// Config holds GA hyperparameters and settings.
type Config struct {
	PopulationSize   int
	Generations      int
	EliteRatio       float64
	MutationRate     float64
	TypeMutationRate float64
	CrossoverRate    float64
	TournamentSize   int
	RandomSeed       int64
}
