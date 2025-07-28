package main

import (
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"uniswapv3-tick-state/abi_instance"
)

type BlockParser interface {
	Output[*BlockReceipt]
	OutputMountable[*BlockEvent]
}

type blockParser struct {
	blockEventReceiver Output[*BlockEvent]
}

func (p *blockParser) PutInput(blockReceipt *BlockReceipt) {
	// no buffer now
	blockEvent := p.ParseBlock(blockReceipt)
	p.blockEventReceiver.PutInput(blockEvent)
}

func (p *blockParser) FinInput() {
	p.blockEventReceiver.FinInput()
}

func (p *blockParser) MountOutput(blockEventReceiver Output[*BlockEvent]) {
	p.blockEventReceiver = blockEventReceiver
}

func NewBlockParser() BlockParser {
	return &blockParser{}
}

func (p *blockParser) ParseBlock(block *BlockReceipt) *BlockEvent {
	events := make([]*Event, 0, 300)

	for _, receipt := range block.Receipts {
		if receipt.Status != 1 {
			continue
		}

		events = append(events, ParseReceipt(receipt)...)
	}

	return &BlockEvent{
		Height: block.Height,
		Events: events,
	}
}

func ParseReceipt(receipt *types.Receipt) []*Event {
	events := make([]*Event, 0, len(receipt.Logs))

	for _, log := range receipt.Logs {
		if len(log.Topics) == 0 {
			continue
		}

		event, err := ParseLog(log)
		if err == nil {
			events = append(events, event)
		}
	}

	return events
}

var (
	ErrUnknownLogTopic = errors.New("unknown log topic")
)

func ParseLog(log *types.Log) (*Event, error) {
	switch log.Topics[0] {
	case abi_instance.MintTopic0:
		return ParseMint(log)
	case abi_instance.BurnTopic0:
		return ParseBurn(log)
	case abi_instance.SwapTopic0:
		return ParseSwap(log)
	default:
		return nil, ErrUnknownLogTopic
	}
}

func ParseMint(log *types.Log) (*Event, error) {
	input, err := ParseInput(log)
	if err != nil {
		return nil, err
	}

	return &Event{
		Address:   log.Address,
		Type:      EventTypeMint,
		TickLower: log.Topics[2].Big(),
		TickUpper: log.Topics[3].Big(),
		Amount:    input[1].(*big.Int),
	}, nil
}

func ParseBurn(log *types.Log) (*Event, error) {
	input, err := ParseInput(log)
	if err != nil {
		return nil, err
	}

	return &Event{
		Address:   log.Address,
		Type:      EventTypeBurn,
		TickLower: log.Topics[2].Big(),
		TickUpper: log.Topics[3].Big(),
		Amount:    input[0].(*big.Int),
	}, nil
}

func ParseSwap(log *types.Log) (*Event, error) {
	input, err := ParseInput(log)
	if err != nil {
		return nil, err
	}

	return &Event{
		Address: log.Address,
		Type:    EventTypeSwap,
		Tick:    input[4].(*big.Int),
	}, nil
}

var (
	ErrParserNotFound     = errors.New("parser not found")
	ErrWrongTopicLen      = errors.New("wrong topic length")
	ErrWrongDataUnpackLen = errors.New("wrong data unpack length")
)

type EventInputParser struct {
	Topic0        common.Hash
	TopicLen      int
	DataUnpackLen int
	ABIEvent      *abi.Event
}

func (p *EventInputParser) Parse(log *types.Log) ([]interface{}, error) {
	if len(log.Topics) != p.TopicLen {
		return nil, ErrWrongTopicLen
	}

	eventInput, err := p.ABIEvent.Inputs.Unpack(log.Data)
	if err != nil {
		return nil, err
	}

	if len(eventInput) != p.DataUnpackLen {
		return nil, ErrWrongDataUnpackLen
	}

	return eventInput, nil
}

var (
	MintEventInputParser = &EventInputParser{
		Topic0:        abi_instance.MintTopic0,
		TopicLen:      4,
		DataUnpackLen: 4,
		ABIEvent:      abi_instance.MintEvent,
	}

	BurnEventInputParser = &EventInputParser{
		Topic0:        abi_instance.BurnTopic0,
		TopicLen:      4,
		DataUnpackLen: 3,
		ABIEvent:      abi_instance.BurnEvent,
	}

	SwapEventInputParser = &EventInputParser{
		Topic0:        abi_instance.SwapTopic0,
		TopicLen:      3,
		DataUnpackLen: 7,
		ABIEvent:      abi_instance.SwapEvent,
	}

	InputParserBook = map[common.Hash]*EventInputParser{
		abi_instance.MintTopic0: MintEventInputParser,
		abi_instance.BurnTopic0: BurnEventInputParser,
		abi_instance.SwapTopic0: SwapEventInputParser,
	}
)

func ParseInput(log *types.Log) ([]interface{}, error) {
	parser, ok := InputParserBook[log.Topics[0]]
	if !ok {
		return nil, ErrParserNotFound
	}

	return parser.Parse(log)
}
