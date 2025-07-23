package randomname

// Default word lists for name generation.
var defaultWords = map[WordType][]string{
	Adjective: {
		"brave", "calm", "eager", "fancy", "gentle", "happy", "jolly", "kind",
		"lively", "nice", "proud", "silly", "witty", "zealous", "mighty", "swift",
		"sharp", "bold", "courageous", "resilient", "daring", "bright", "creative",
		"innovative", "dynamic", "energetic", "vibrant", "radiant", "sincere", "honest",
		"steadfast", "ardent", "spirited", "graceful", "gritty", "focused", "optimistic",
		"robust", "stalwart", "resolute", "vigorous", "agile", "ambitious", "ancient",
		"artistic", "authentic", "balanced", "brilliant", "charming", "cheerful", "clever",
		"confident", "cosmic", "crisp", "curious", "dazzling", "determined", "diligent",
		"elegant", "enchanted", "epic", "fearless", "fierce", "flexible", "flowing",
		"friendly", "frosty", "gallant", "generous", "gleaming", "glorious", "golden",
		"harmonious", "heroic", "humble", "illustrious", "immense", "incredible", "inspired",
		"intelligent", "intrepid", "legendary", "luminous", "majestic", "marvelous", "mindful",
		"modern", "mystical", "noble", "peaceful", "persistent", "playful", "polished",
		"powerful", "precious", "pristine", "quick", "quirky", "radiant", "refreshing",
		"remarkable", "royal", "sage", "savvy", "serene", "shining", "skillful",
		"sleek", "smooth", "sophisticated", "sparkling", "spectacular", "splendid", "stellar",
		"strong", "stunning", "sublime", "subtle", "sunny", "super", "supreme",
		"tactical", "talented", "tenacious", "thoughtful", "thriving", "tidy", "tranquil",
		"trusty", "ultimate", "unique", "valiant", "versatile", "vivid", "warm",
		"whimsical", "wise", "wonderful", "worthy", "youthful", "zesty", "zippy",
	},

	Noun: {
		"squirrel", "tiger", "eagle", "dolphin", "panther", "lion", "panda", "koala",
		"whale", "shark", "wolf", "falcon", "otter", "rabbit", "bear", "fox", "hedgehog",
		"owl", "leopard", "cheetah", "hyena", "buffalo", "zebra", "giraffe", "coyote",
		"raccoon", "badger", "moose", "stallion", "gazelle", "mongoose", "cougar", "jaguar",
		"bison", "viper", "python", "cobra", "lizard", "frog", "beaver", "porcupine",
		"skunk", "antelope", "hamster", "gerbil", "alpaca", "armadillo", "barracuda",
		"beetle", "bobcat", "butterfly", "camel", "canary", "cardinal", "caribou",
		"cassowary", "chameleon", "chinchilla", "chipmunk", "condor", "cormorant", "crab",
		"crane", "cricket", "crocodile", "crow", "deer", "dingo", "dragonfly",
		"duck", "elephant", "elk", "emu", "ferret", "finch", "firefly",
		"flamingo", "gecko", "goose", "gorilla", "grasshopper", "hawk", "heron",
		"hippo", "horse", "hummingbird", "iguana", "impala", "jackal", "jellyfish",
		"kangaroo", "kestrel", "kingfisher", "kiwi", "ladybug", "lemur", "llama",
		"lobster", "lynx", "macaw", "magpie", "mammoth", "manatee", "manta",
		"marlin", "meerkat", "monkey", "narwhal", "newt", "octopus", "ocelot",
		"okapi", "orangutan", "orca", "oriole", "osprey", "ostrich", "oyster",
		"parrot", "peacock", "pelican", "penguin", "phoenix", "platypus", "puma",
		"quail", "quokka", "raven", "reindeer", "rhino", "robin", "rooster",
		"salamander", "salmon", "scorpion", "seagull", "seahorse", "seal", "sparrow",
		"spider", "squid", "starfish", "stingray", "swan", "tapir", "toucan",
		"trout", "tuna", "turkey", "turtle", "unicorn", "walrus", "warthog",
		"wasp", "weasel", "woodpecker", "wombat", "yak", "yellowfin", "zebu",
	},

	Color: {
		// Basic colors
		"red", "blue", "green", "yellow", "orange", "purple", "black", "white", "gray", "brown",
		// Extended colors
		"crimson", "azure", "emerald", "golden", "silver", "violet", "magenta", "cyan", "amber", "indigo",
		// Nature-inspired
		"coral", "jade", "ruby", "sapphire", "pearl", "ebony", "ivory", "bronze", "copper", "rose",
		// Modern/tech colors
		"neon", "quantum", "cyber", "plasma", "electric", "atomic", "cosmic", "stellar", "chrome", "titanium",
	},

	Size: {
		// Basic sizes
		"tiny", "small", "little", "medium", "large", "big", "huge", "giant", "massive", "colossal",
		// Tech-inspired sizes
		"mini", "micro", "nano", "mega", "giga", "ultra", "super", "hyper",
		// Additional sizes
		"petite", "compact", "grand", "enormous", "immense", "vast", "titanic",
	},

	Origin: {
		// Geographic origins
		"arctic", "tropical", "desert", "mountain", "forest", "ocean", "river", "lake", "island", "valley",
		// Celestial origins
		"lunar", "solar", "cosmic", "stellar", "celestial", "galactic", "astral", "meteoric",
		// Environmental origins
		"urban", "rural", "coastal", "highland", "lowland", "polar", "equatorial",
		// Additional origins
		"alpine", "tundra", "savanna", "prairie", "volcanic",
	},

	Action: {
		// Movement actions
		"flying", "running", "jumping", "swimming", "diving", "soaring", "gliding", "racing",
		// Performance actions
		"dancing", "singing", "hunting", "prowling", "stalking", "leaping", "climbing", "sprinting",
		// Energy actions
		"blazing", "shining", "sparkling", "glowing", "radiating", "pulsing", "flickering", "beaming",
		// Additional actions
		"dashing", "zooming", "hovering", "floating", "wandering", "exploring", "charging", "surging",
		"whirling", "spinning", "tumbling", "vaulting",
	},
}

// getWords returns the word list for a given type, merging custom words if provided.
func getWords(wordType WordType, customWords map[WordType][]string) []string {
	words := defaultWords[wordType]
	if custom, ok := customWords[wordType]; ok && len(custom) > 0 {
		// Create a new slice to avoid modifying the default
		merged := make([]string, 0, len(words)+len(custom))
		merged = append(merged, words...)
		merged = append(merged, custom...)
		return merged
	}
	return words
}
