package chainreg

// ChainCode is an enum-like structure for keeping track of the chains
// currently supported within lnd.
type ChainCode uint32

const (
	// BitcoinChain is Bitcoin's chain.
	BitcoinChain ChainCode = iota

	// MonacoinChain is Monacoin's chain.
	MonacoinChain
)

// String returns a string representation of the target ChainCode.
func (c ChainCode) String() string {
	switch c {
	case BitcoinChain:
		return "bitcoin"
	case MonacoinChain:
		return "Monacoin"
	default:
		return "kekcoin"
	}
}
