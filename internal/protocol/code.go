package protocol

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
)

var codePattern = regexp.MustCompile(`^[a-z]+-[a-z]+-\d{2}$`)

var adjectives = [256]string{
	"able", "acid", "aged", "airy", "arch", "avid", "back", "bald",
	"bare", "base", "bent", "best", "bird", "bite", "blip", "blue",
	"bold", "bone", "boom", "boss", "boxy", "bulk", "bump", "burn",
	"busy", "calm", "cave", "chip", "chop", "city", "clan", "clay",
	"clip", "club", "coal", "coat", "code", "coil", "cold", "cool",
	"cope", "copy", "cord", "core", "cost", "cozy", "crew", "crop",
	"cure", "curl", "cute", "dale", "damp", "dark", "dart", "dawn",
	"dear", "deep", "deer", "demo", "dent", "dial", "dice", "dime",
	"dire", "dirt", "disc", "dock", "dome", "done", "doom", "door",
	"dose", "dove", "down", "draw", "drip", "drop", "drum", "dual",
	"dull", "dune", "dusk", "dust", "each", "earl", "earn", "ease",
	"east", "easy", "edge", "epic", "even", "ever", "evil", "exam",
	"exit", "expo", "face", "fact", "fade", "fair", "fake", "fame",
	"fang", "farm", "fast", "fate", "fawn", "fern", "film", "find",
	"fine", "fire", "firm", "fish", "fist", "five", "flag", "flame",
	"flat", "flex", "flip", "flow", "foam", "fold", "fond", "font",
	"food", "ford", "fork", "fort", "foul", "four", "free", "frog",
	"from", "fuel", "full", "fund", "fury", "fuse", "fuzz", "gain",
	"gale", "game", "gate", "gear", "gift", "glad", "glow", "glue",
	"goat", "gold", "golf", "gone", "good", "grab", "gray", "grid",
	"grim", "grip", "grit", "grow", "gust", "gyre", "hail", "hale",
	"half", "hall", "halt", "halo", "hard", "harp", "haze", "heap",
	"heat", "herd", "hero", "high", "hike", "hill", "hilt", "hint",
	"hold", "home", "hood", "hook", "hope", "host", "howl", "huge",
	"hull", "hump", "hush", "hymn", "icon", "idle", "inch", "into",
	"iron", "isle", "item", "jade", "jail", "jazz", "jest", "jive",
	"join", "joke", "jolt", "jump", "jury", "just", "keen", "kelp",
	"kept", "kick", "kind", "king", "kite", "knit", "knob", "knot",
	"know", "lace", "lake", "lamb", "lamp", "lane", "lark", "last",
	"late", "lawn", "lead", "leaf", "lean", "left", "less", "lick",
	"lift", "like", "lime", "limp", "line", "link", "lion", "live",
	"lock", "loft", "long", "loop", "loom", "loud", "love", "luck",
}

var nouns = [256]string{
	"acorn", "agate", "alpha", "amber", "angel", "anvil", "apple", "arrow",
	"aspen", "atlas", "badge", "basin", "beach", "beast", "berry", "birch",
	"blade", "blaze", "bloom", "board", "brace", "brain", "brave", "bread",
	"brick", "brook", "brush", "camel", "candy", "cargo", "cedar", "chain",
	"chalk", "charm", "chess", "chief", "cider", "cigar", "cliff", "cloud",
	"coach", "cobra", "coral", "comet", "crane", "crash", "cream", "creek",
	"crest", "crown", "dance", "delta", "depot", "digit", "dingo", "disco",
	"draft", "drain", "drake", "dream", "drift", "drill", "drive", "drone",
	"dwarf", "eagle", "earth", "ember", "entry", "envoy", "epoch", "equal",
	"event", "exile", "fable", "fault", "feast", "fiber", "field", "finch",
	"fjord", "flame", "flask", "fleet", "flint", "flood", "flour", "flute",
	"focal", "forge", "forum", "frame", "frost", "fruit", "ghost", "giant",
	"glade", "glass", "globe", "grace", "grain", "grape", "grass", "grove",
	"guard", "guide", "haven", "hawk", "heart", "hedge", "heron", "honey",
	"horse", "hotel", "house", "hyena", "image", "index", "ivory", "jewel",
	"joker", "juice", "kayak", "knife", "knock", "lance", "latch", "lemon",
	"level", "light", "lilac", "linen", "llama", "lotus", "manor", "maple",
	"march", "marsh", "medal", "melon", "metal", "minor", "mirth", "model",
	"moose", "motor", "mound", "mouse", "mouth", "mural", "nerve", "night",
	"noble", "north", "novel", "oasis", "ocean", "olive", "onion", "opera",
	"orbit", "otter", "oxide", "panda", "panel", "patch", "pearl", "pedal",
	"penny", "perch", "phase", "piano", "pilot", "pixel", "place", "plank",
	"plant", "plume", "point", "polar", "poppy", "pouch", "power", "press",
	"pride", "prism", "probe", "proxy", "pulse", "quail", "query", "quest",
	"radar", "raven", "realm", "ridge", "rider", "rivet", "robin", "rover",
	"saint", "scale", "scene", "scout", "shade", "shark", "shell", "shine",
	"shore", "sigma", "siren", "slate", "slope", "smoke", "snail", "solar",
	"sonic", "spark", "spear", "spice", "spine", "spray", "squad", "staff",
	"stage", "stake", "steam", "steel", "steep", "stone", "storm", "stove",
	"sugar", "surge", "swamp", "swift", "sword", "tiger", "timber", "torch",
	"tower", "trail", "trout", "tulip", "vapor", "vault", "venom", "verse",
	"vigor", "vinyl", "viper", "vivid", "watch", "whale", "wheel", "wraith",
}

func GenerateCode() string {
	adj := adjectives[randInt(256)]
	noun := nouns[randInt(256)]
	num := randInt(100)
	return fmt.Sprintf("%s-%s-%02d", adj, noun, num)
}

func ValidateCode(code string) bool {
	return codePattern.MatchString(code)
}

func randInt(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return int(n.Int64())
}
