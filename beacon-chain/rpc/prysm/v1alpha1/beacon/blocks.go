package beacon

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/api/pagination"
	"github.com/prysmaticlabs/prysm/async/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/time/slots"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// blockContainer represents an instance of
// block along with its relevant metadata.
type blockContainer struct {
	blk         interfaces.SignedBeaconBlock
	root        [32]byte
	isCanonical bool
}

// ListBlocks retrieves blocks by root, slot, or epoch.
//
// The server may return multiple blocks in the case that a slot or epoch is
// provided as the filter criteria. The server may return an empty list when
// no blocks in their database match the filter criteria. This RPC should
// not return NOT_FOUND. Only one filter criteria should be used.
func (bs *Server) ListBlocks(
	ctx context.Context, req *ethpb.ListBlocksRequest,
) (*ethpb.ListBlocksResponse, error) {
	ctrs, numBlks, nextPageToken, err := bs.listBlocks(ctx, req)
	if err != nil {
		return nil, err
	}
	blkContainers, err := convertToProto(ctrs)
	if err != nil {
		return nil, err
	}

	return &ethpb.ListBlocksResponse{
		BlockContainers: blkContainers,
		TotalSize:       int32(numBlks),
		NextPageToken:   nextPageToken,
	}, nil
}

// ListBeaconBlocks retrieves blocks by root, slot, or epoch.
//
// The server may return multiple blocks in the case that a slot or epoch is
// provided as the filter criteria. The server may return an empty list when
// no blocks in their database match the filter criteria. This RPC should
// not return NOT_FOUND. Only one filter criteria should be used.
func (bs *Server) ListBeaconBlocks(
	ctx context.Context, req *ethpb.ListBlocksRequest,
) (*ethpb.ListBeaconBlocksResponse, error) {
	ctrs, numBlks, nextPageToken, err := bs.listBlocks(ctx, req)
	if err != nil {
		return nil, err
	}
	altCtrs, err := convertFromV1Containers(ctrs)
	if err != nil {
		return nil, err
	}
	return &ethpb.ListBeaconBlocksResponse{
		BlockContainers: altCtrs,
		TotalSize:       int32(numBlks),
		NextPageToken:   nextPageToken,
	}, nil
}

func (bs *Server) listBlocks(ctx context.Context, req *ethpb.ListBlocksRequest) ([]blockContainer, int, string, error) {
	if int(req.PageSize) > cmd.Get().MaxRPCPageSize {
		return nil, 0, "", status.Errorf(codes.InvalidArgument, "Requested page size %d can not be greater than max size %d",
			req.PageSize, cmd.Get().MaxRPCPageSize)
	}

	switch q := req.QueryFilter.(type) {
	case *ethpb.ListBlocksRequest_Epoch:
		return bs.listBlocksForEpoch(ctx, req, q)
	case *ethpb.ListBlocksRequest_Root:
		return bs.listBlocksForRoot(ctx, req, q)
	case *ethpb.ListBlocksRequest_Slot:
		return bs.listBlocksForSlot(ctx, req, q)
	case *ethpb.ListBlocksRequest_Genesis:
		return bs.listBlocksForGenesis(ctx, req, q)
	default:
		return nil, 0, "", status.Errorf(codes.InvalidArgument, "Must specify a filter criteria for fetching blocks. Criteria %T not supported", q)
	}
}

func convertFromV1Containers(ctrs []blockContainer) ([]*ethpb.BeaconBlockContainer, error) {
	protoCtrs := make([]*ethpb.BeaconBlockContainer, len(ctrs))
	var err error
	for i, c := range ctrs {
		protoCtrs[i], err = convertToBlockContainer(c.blk, c.root, c.isCanonical)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get block container: %v", err)
		}
	}
	return protoCtrs, nil
}

func convertToBlockContainer(blk interfaces.SignedBeaconBlock, root [32]byte, isCanonical bool) (*ethpb.BeaconBlockContainer, error) {
	ctr := &ethpb.BeaconBlockContainer{
		BlockRoot: root[:],
		Canonical: isCanonical,
	}

	switch blk.Version() {
	case version.Phase0:
		rBlk, err := blk.PbPhase0Block()
		if err != nil {
			return nil, err
		}
		ctr.Block = &ethpb.BeaconBlockContainer_Phase0Block{Phase0Block: rBlk}
	case version.Altair:
		rBlk, err := blk.PbAltairBlock()
		if err != nil {
			return nil, err
		}
		ctr.Block = &ethpb.BeaconBlockContainer_AltairBlock{AltairBlock: rBlk}
	case version.Bellatrix:
		rBlk, err := blk.PbBellatrixBlock()
		if err != nil {
			return nil, err
		}
		ctr.Block = &ethpb.BeaconBlockContainer_BellatrixBlock{BellatrixBlock: rBlk}
	default:
		return nil, errors.Errorf("block type is not recognized: %d", blk.Version())
	}
	return ctr, nil
}

// listBlocksForEpoch retrieves all blocks for the provided epoch.
func (bs *Server) listBlocksForEpoch(ctx context.Context, req *ethpb.ListBlocksRequest, q *ethpb.ListBlocksRequest_Epoch) ([]blockContainer, int, string, error) {
	blks, _, err := bs.BeaconDB.Blocks(ctx, filters.NewFilter().SetStartEpoch(q.Epoch).SetEndEpoch(q.Epoch))
	if err != nil {
		return nil, 0, strconv.Itoa(0), status.Errorf(codes.Internal, "Could not get blocks: %v", err)
	}

	numBlks := len(blks)
	if len(blks) == 0 {
		return []blockContainer{}, numBlks, strconv.Itoa(0), nil
	}

	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), numBlks)
	if err != nil {
		return nil, 0, strconv.Itoa(0), status.Errorf(codes.Internal, "Could not paginate blocks: %v", err)
	}

	returnedBlks := blks[start:end]
	containers := make([]blockContainer, len(returnedBlks))
	for i, b := range returnedBlks {
		root, err := b.Block().HashTreeRoot()
		if err != nil {
			return nil, 0, strconv.Itoa(0), err
		}
		canonical, err := bs.CanonicalFetcher.IsCanonical(ctx, root)
		if err != nil {
			return nil, 0, strconv.Itoa(0), status.Errorf(codes.Internal, "Could not determine if block is canonical: %v", err)
		}
		containers[i] = blockContainer{
			blk:         b,
			root:        root,
			isCanonical: canonical,
		}
	}

	return containers, numBlks, nextPageToken, nil
}

// listBlocksForRoot retrieves the block for the provided root.
func (bs *Server) listBlocksForRoot(ctx context.Context, _ *ethpb.ListBlocksRequest, q *ethpb.ListBlocksRequest_Root) ([]blockContainer, int, string, error) {
	blk, err := bs.BeaconDB.Block(ctx, bytesutil.ToBytes32(q.Root))
	if err != nil {
		return nil, 0, strconv.Itoa(0), status.Errorf(codes.Internal, "Could not retrieve block: %v", err)
	}
	if blk == nil || blk.IsNil() {
		return []blockContainer{}, 0, strconv.Itoa(0), nil
	}
	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, 0, strconv.Itoa(0), status.Errorf(codes.Internal, "Could not determine block root: %v", err)
	}
	canonical, err := bs.CanonicalFetcher.IsCanonical(ctx, root)
	if err != nil {
		return nil, 0, strconv.Itoa(0), status.Errorf(codes.Internal, "Could not determine if block is canonical: %v", err)
	}
	return []blockContainer{{
		blk:         blk,
		root:        root,
		isCanonical: canonical,
	}}, 1, strconv.Itoa(0), nil
}

// listBlocksForSlot retrieves all blocks for the provided slot.
func (bs *Server) listBlocksForSlot(ctx context.Context, req *ethpb.ListBlocksRequest, q *ethpb.ListBlocksRequest_Slot) ([]blockContainer, int, string, error) {
	hasBlocks, blks, err := bs.BeaconDB.BlocksBySlot(ctx, q.Slot)
	if err != nil {
		return nil, 0, strconv.Itoa(0), status.Errorf(codes.Internal, "Could not retrieve blocks for slot %d: %v", q.Slot, err)
	}
	if !hasBlocks {
		return []blockContainer{}, 0, strconv.Itoa(0), nil
	}

	numBlks := len(blks)

	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), numBlks)
	if err != nil {
		return nil, 0, strconv.Itoa(0), status.Errorf(codes.Internal, "Could not paginate blocks: %v", err)
	}

	returnedBlks := blks[start:end]
	containers := make([]blockContainer, len(returnedBlks))
	for i, b := range returnedBlks {
		root, err := b.Block().HashTreeRoot()
		if err != nil {
			return nil, 0, strconv.Itoa(0), status.Errorf(codes.Internal, "Could not determine block root: %v", err)
		}
		canonical, err := bs.CanonicalFetcher.IsCanonical(ctx, root)
		if err != nil {
			return nil, 0, strconv.Itoa(0), status.Errorf(codes.Internal, "Could not determine if block is canonical: %v", err)
		}
		containers[i] = blockContainer{
			blk:         b,
			root:        root,
			isCanonical: canonical,
		}
	}
	return containers, numBlks, nextPageToken, nil
}

// listBlocksForGenesis retrieves the genesis block.
func (bs *Server) listBlocksForGenesis(ctx context.Context, _ *ethpb.ListBlocksRequest, _ *ethpb.ListBlocksRequest_Genesis) ([]blockContainer, int, string, error) {
	genBlk, err := bs.BeaconDB.GenesisBlock(ctx)
	if err != nil {
		return nil, 0, strconv.Itoa(0), status.Errorf(codes.Internal, "Could not retrieve blocks for genesis slot: %v", err)
	}
	if err := wrapper.BeaconBlockIsNil(genBlk); err != nil {
		return []blockContainer{}, 0, strconv.Itoa(0), status.Errorf(codes.NotFound, "Could not find genesis block: %v", err)
	}
	root, err := genBlk.Block().HashTreeRoot()
	if err != nil {
		return nil, 0, strconv.Itoa(0), status.Errorf(codes.Internal, "Could not determine block root: %v", err)
	}
	return []blockContainer{{
		blk:         genBlk,
		root:        root,
		isCanonical: true,
	}}, 1, strconv.Itoa(0), nil
}

func convertToProto(ctrs []blockContainer) ([]*ethpb.BeaconBlockContainer, error) {
	protoCtrs := make([]*ethpb.BeaconBlockContainer, len(ctrs))
	for i, c := range ctrs {
		phBlk, err := c.blk.PbPhase0Block()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get phase 0 block: %v", err)
		}
		copiedRoot := c.root
		protoCtrs[i] = &ethpb.BeaconBlockContainer{
			Block:     &ethpb.BeaconBlockContainer_Phase0Block{Phase0Block: phBlk},
			BlockRoot: copiedRoot[:],
			Canonical: c.isCanonical,
		}
	}
	return protoCtrs, nil
}

// GetChainHead retrieves information about the head of the beacon chain from
// the view of the beacon chain node.
//
// This includes the head block slot and root as well as information about
// the most recent finalized and justified slots.
func (bs *Server) GetChainHead(ctx context.Context, _ *emptypb.Empty) (*ethpb.ChainHead, error) {
	return bs.chainHeadRetrieval(ctx)
}

// StreamBlocks to clients every single time a block is received by the beacon node.
func (bs *Server) StreamBlocks(req *ethpb.StreamBlocksRequest, stream ethpb.BeaconChain_StreamBlocksServer) error {
	blocksChannel := make(chan *feed.Event, 1)
	var blockSub event.Subscription
	if req.VerifiedOnly {
		blockSub = bs.StateNotifier.StateFeed().Subscribe(blocksChannel)
	} else {
		blockSub = bs.BlockNotifier.BlockFeed().Subscribe(blocksChannel)
	}
	defer blockSub.Unsubscribe()

	for {
		select {
		case blockEvent := <-blocksChannel:
			if req.VerifiedOnly {
				if blockEvent.Type == statefeed.BlockProcessed {
					data, ok := blockEvent.Data.(*statefeed.BlockProcessedData)
					if !ok || data == nil {
						continue
					}
					phBlk, err := data.SignedBlock.PbPhase0Block()
					if err != nil {
						log.Error(err)
						continue
					}
					if err := stream.Send(phBlk); err != nil {
						return status.Errorf(codes.Unavailable, "Could not send over stream: %v", err)
					}
				}
			} else {
				if blockEvent.Type == blockfeed.ReceivedBlock {
					data, ok := blockEvent.Data.(*blockfeed.ReceivedBlockData)
					if !ok {
						// Got bad data over the stream.
						continue
					}
					if data.SignedBlock == nil {
						// One nil block shouldn't stop the stream.
						continue
					}
					headState, err := bs.HeadFetcher.HeadState(bs.Ctx)
					if err != nil {
						log.WithError(err).WithField("blockSlot", data.SignedBlock.Block().Slot()).Error("Could not get head state")
						continue
					}
					signed := data.SignedBlock
					if err := blocks.VerifyBlockSignature(headState, signed.Block().ProposerIndex(), signed.Signature(), signed.Block().HashTreeRoot); err != nil {
						log.WithError(err).WithField("blockSlot", data.SignedBlock.Block().Slot()).Error("Could not verify block signature")
						continue
					}
					phBlk, err := signed.PbPhase0Block()
					if err != nil {
						log.Error(err)
						continue
					}
					if err := stream.Send(phBlk); err != nil {
						return status.Errorf(codes.Unavailable, "Could not send over stream: %v", err)
					}
				}
			}
		case <-blockSub.Err():
			return status.Error(codes.Aborted, "Subscriber closed, exiting goroutine")
		case <-bs.Ctx.Done():
			return status.Error(codes.Canceled, "Context canceled")
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "Context canceled")
		}
	}
}

// StreamChainHead to clients every single time the head block and state of the chain change.
func (bs *Server) StreamChainHead(_ *emptypb.Empty, stream ethpb.BeaconChain_StreamChainHeadServer) error {
	stateChannel := make(chan *feed.Event, 1)
	stateSub := bs.StateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()
	for {
		select {
		case stateEvent := <-stateChannel:
			if stateEvent.Type == statefeed.BlockProcessed {
				res, err := bs.chainHeadRetrieval(stream.Context())
				if err != nil {
					return status.Errorf(codes.Internal, "Could not retrieve chain head: %v", err)
				}
				if err := stream.Send(res); err != nil {
					return status.Errorf(codes.Unavailable, "Could not send over stream: %v", err)
				}
			}
		case <-stateSub.Err():
			return status.Error(codes.Aborted, "Subscriber closed, exiting goroutine")
		case <-bs.Ctx.Done():
			return status.Error(codes.Canceled, "Context canceled")
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "Context canceled")
		}
	}
}

// Retrieve chain head information from the DB and the current beacon state.
func (bs *Server) chainHeadRetrieval(ctx context.Context) (*ethpb.ChainHead, error) {
	headBlock, err := bs.HeadFetcher.HeadBlock(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head block")
	}
	if err := wrapper.BeaconBlockIsNil(headBlock); err != nil {
		return nil, status.Errorf(codes.NotFound, "Head block of chain was nil: %v", err)
	}
	headBlockRoot, err := headBlock.Block().HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head block root: %v", err)
	}

	validGenesis := false
	validateCP := func(cp *ethpb.Checkpoint, name string) error {
		if bytesutil.ToBytes32(cp.Root) == params.BeaconConfig().ZeroHash && cp.Epoch == 0 {
			if validGenesis {
				return nil
			}
			// Retrieve genesis block in the event we have genesis checkpoints.
			genBlock, err := bs.BeaconDB.GenesisBlock(ctx)
			if err != nil || wrapper.BeaconBlockIsNil(genBlock) != nil {
				return status.Error(codes.Internal, "Could not get genesis block")
			}
			validGenesis = true
			return nil
		}
		b, err := bs.BeaconDB.Block(ctx, bytesutil.ToBytes32(cp.Root))
		if err != nil {
			return status.Errorf(codes.Internal, "Could not get %s block: %v", name, err)
		}
		if err := wrapper.BeaconBlockIsNil(b); err != nil {
			return status.Errorf(codes.Internal, "Could not get %s block: %v", name, err)
		}
		return nil
	}

	finalizedCheckpoint, err := bs.FinalizationFetcher.FinalizedCheckpt()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get finalized checkpoint: %v", err)
	}
	if err := validateCP(finalizedCheckpoint, "finalized"); err != nil {
		return nil, err
	}

	justifiedCheckpoint, err := bs.FinalizationFetcher.CurrentJustifiedCheckpt()
	if err != nil {
		return nil, err
	}
	if err := validateCP(justifiedCheckpoint, "justified"); err != nil {
		return nil, err
	}

	prevJustifiedCheckpoint, err := bs.FinalizationFetcher.PreviousJustifiedCheckpt()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get previous justified checkpoint: %v", err)
	}
	if err := validateCP(prevJustifiedCheckpoint, "prev justified"); err != nil {
		return nil, err
	}

	fSlot, err := slots.EpochStart(finalizedCheckpoint.Epoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get epoch start slot from finalized checkpoint epoch")
	}
	jSlot, err := slots.EpochStart(justifiedCheckpoint.Epoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get epoch start slot from justified checkpoint epoch")
	}
	pjSlot, err := slots.EpochStart(prevJustifiedCheckpoint.Epoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get epoch start slot from prev justified checkpoint epoch")
	}
	return &ethpb.ChainHead{
		HeadSlot:                   headBlock.Block().Slot(),
		HeadEpoch:                  slots.ToEpoch(headBlock.Block().Slot()),
		HeadBlockRoot:              headBlockRoot[:],
		FinalizedSlot:              fSlot,
		FinalizedEpoch:             finalizedCheckpoint.Epoch,
		FinalizedBlockRoot:         finalizedCheckpoint.Root,
		JustifiedSlot:              jSlot,
		JustifiedEpoch:             justifiedCheckpoint.Epoch,
		JustifiedBlockRoot:         justifiedCheckpoint.Root,
		PreviousJustifiedSlot:      pjSlot,
		PreviousJustifiedEpoch:     prevJustifiedCheckpoint.Epoch,
		PreviousJustifiedBlockRoot: prevJustifiedCheckpoint.Root,
	}, nil
}
