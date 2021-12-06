// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package istanbul

import (
	"math/big"
	"sync"

	"github.com/naoina/toml"
)

type ProposerPolicyId uint64

const (
	RoundRobin ProposerPolicyId = iota
	Sticky
)

// ProposerPolicy represents the Validator Proposer Policy
type ProposerPolicy struct {
	Id         ProposerPolicyId    // Could be RoundRobin or Sticky
	By         ValidatorSortByFunc // func that defines how the ValidatorSet should be sorted
	registry   []ValidatorSet      // Holds the ValidatorSet for a given block height
	registryMU *sync.Mutex         // Mutex to lock access to changes to Registry
}

// NewRoundRobinProposerPolicy returns a RoundRobin ProposerPolicy with ValidatorSortByString as default sort function
func NewRoundRobinProposerPolicy() *ProposerPolicy {
	return NewProposerPolicy(RoundRobin)
}

// NewStickyProposerPolicy return a Sticky ProposerPolicy with ValidatorSortByString as default sort function
func NewStickyProposerPolicy() *ProposerPolicy {
	return NewProposerPolicy(Sticky)
}

func NewProposerPolicy(id ProposerPolicyId) *ProposerPolicy {
	return NewProposerPolicyByIdAndSortFunc(id, ValidatorSortByString())
}

func NewProposerPolicyByIdAndSortFunc(id ProposerPolicyId, by ValidatorSortByFunc) *ProposerPolicy {
	return &ProposerPolicy{Id: id, By: by, registryMU: new(sync.Mutex)}
}

type proposerPolicyToml struct {
	Id ProposerPolicyId
}

func (p *ProposerPolicy) MarshalTOML() ([]byte, error) {
	pp := &proposerPolicyToml{Id: p.Id}
	return toml.Marshal(pp)
}

func (p *ProposerPolicy) UnmarshalTOML(input []byte) error {
	var pp proposerPolicyToml
	err := toml.Unmarshal(input, &pp)
	if err != nil {
		return err
	}
	p.Id = pp.Id
	p.By = ValidatorSortByString()
	return nil
}

// Use sets the ValidatorSortByFunc for the given ProposerPolicy and sorts the validatorSets according to it
func (p *ProposerPolicy) Use(v ValidatorSortByFunc) {
	p.By = v

	for _, validatorSet := range p.registry {
		validatorSet.SortValidators()
	}
}

// RegisterValidatorSet stores the given ValidatorSet in the policy registry
func (p *ProposerPolicy) RegisterValidatorSet(valSet ValidatorSet) {
	p.registryMU.Lock()
	defer p.registryMU.Unlock()

	if len(p.registry) == 0 {
		p.registry = []ValidatorSet{valSet}
	} else {
		p.registry = append(p.registry, valSet)
	}
}

// ClearRegistry removes any ValidatorSet from the ProposerPolicy registry
func (p *ProposerPolicy) ClearRegistry() {
	p.registryMU.Lock()
	defer p.registryMU.Unlock()

	p.registry = nil
}

type Config struct {
	// 每个IBFT或 QBFT回合的最小请求超时（以毫秒为单位）。请求超时是如果前一轮没有完成，IBFT 触发新一轮的超时时间。随着超时被更频繁地命中，此时间段会增加。
	RequestTimeout         uint64          `toml:",omitempty"` // The timeout for each Istanbul round in milliseconds.
	// 出块时间 (两个连续块的时间戳之间的默认最小差异（以秒为单位）)
	BlockPeriod            uint64          `toml:",omitempty"` // Default minimum difference between two consecutive block's timestamps in second
	ProposerPolicy         *ProposerPolicy `toml:",omitempty"` // The policy for proposer selection
	// 检查点和重置未决投票之前的块数
	Epoch                  uint64          `toml:",omitempty"` // The number of blocks after which to checkpoint and reset the pending votes
	Ceil2Nby3Block         *big.Int        `toml:",omitempty"` // Number of confirmations required to move from one state to next [2F + 1 to Ceil(2N/3)]
	// 在它们被认为是未来的块之前允许块的当前时间的最长时间（以秒为单位）
	// 从当前时间开始，块被视为未来块之前允许的最长时间，以秒为单位。这允许节点稍微不同步而不会收到“未来挖掘太远”消息。默认值为 0。
	AllowedFutureBlockTime uint64          `toml:",omitempty"` // Max time (in seconds) from current time allowed for blocks, before they're considered future blocks
	TestQBFTBlock          *big.Int        `toml:",omitempty"` // Fork block at which block confirmations are done using qbft consensus instead of ibft
}

var DefaultConfig = &Config{
	RequestTimeout:         10000,
	BlockPeriod:            1,
	ProposerPolicy:         NewRoundRobinProposerPolicy(),
	Epoch:                  30000,
	Ceil2Nby3Block:         big.NewInt(0),
	AllowedFutureBlockTime: 0,
	TestQBFTBlock:          big.NewInt(0),
}

// QBFTBlockNumber returns the qbftBlock fork block number, returns -1 if qbftBlock is not defined
func (c Config) QBFTBlockNumber() int64 {
	if c.TestQBFTBlock == nil {
		return -1
	}
	return c.TestQBFTBlock.Int64()
}

// IsQBFTConsensusAt checks if qbft consensus is enabled for the block height identified by the given header
func (c *Config) IsQBFTConsensusAt(blockNumber *big.Int) bool {
	// If qbftBlock is not defined in genesis qbft consensus is not used
	if c.TestQBFTBlock == nil {
		return false
	}

	if c.TestQBFTBlock.Uint64() == 0 {
		return true
	}

	if blockNumber.Cmp(c.TestQBFTBlock) >= 0 {
		return true
	}
	return false
}
