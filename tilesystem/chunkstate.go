package tilesystem

import (
	"ebittest/utils"
	"log"
)

type chunkState uint8

const (
	ChunkState_Unloaded chunkState = iota
	ChunkState_Loading
	ChunkState_Active
	ChunkState_Evicting
)

func canTransitionChunkState(from, to chunkState) bool {
	if from == to {
		return true
	}

	switch from {
	case ChunkState_Unloaded:
		return to == ChunkState_Loading
	case ChunkState_Loading:
		return to == ChunkState_Unloaded || to == ChunkState_Active
	case ChunkState_Active:
		return to == ChunkState_Evicting || to == ChunkState_Unloaded
	case ChunkState_Evicting:
		return to == ChunkState_Unloaded || to == ChunkState_Active
	default:
		return false
	}
}

func (cc *ChunkContainer) setState(pos utils.Vec2i, next chunkState) {
	meta, ok := cc.chunkMeta[pos]
	if !ok {
		meta = &chunkLoadMeta{state: ChunkState_Unloaded}
		cc.chunkMeta[pos] = meta
	}

	if !canTransitionChunkState(meta.state, next) {
		log.Printf("invalid chunk state transition at %v: %v -> %v", pos, meta.state, next)
	}

	if next == ChunkState_Unloaded {
		delete(cc.chunkMeta, pos)
	} else {
		meta.state = next
	}
}
