package ballots

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/spacemeshos/go-spacemesh/codec"
	"github.com/spacemeshos/go-spacemesh/common/types"
	"github.com/spacemeshos/go-spacemesh/sql"
)

func decodeBallot(id types.BallotID, pubkey, body *bytes.Reader, malicious bool) (*types.Ballot, error) {
	size := pubkey.Len()
	pubkeyBytes := make([]byte, size)
	if n, err := pubkey.Read(pubkeyBytes); err != nil {
		if err != io.EOF {
			return nil, fmt.Errorf("copy pubkey: %w", err)
		}
	} else if n != size {
		return nil, errors.New("public key data missing")
	}
	ballot := types.Ballot{}
	if n, err := codec.DecodeFrom(body, &ballot); err != nil {
		if err != io.EOF {
			return nil, fmt.Errorf("decode body of the %s: %w", id, err)
		}
	} else if n == 0 {
		return nil, errors.New("ballot data missing")
	}
	ballot.SetID(id)
	ballot.SetSmesherID(types.BytesToNodeID(pubkeyBytes))
	if malicious {
		ballot.SetMalicious()
	}
	return &ballot, nil
}

// Add ballot to the database.
func Add(db sql.Executor, ballot *types.Ballot) error {
	bytes, err := codec.Encode(ballot)
	if err != nil {
		return fmt.Errorf("encode ballot %s: %w", ballot.ID(), err)
	}
	if _, err := db.Exec(`insert into ballots 
		(id, layer, pubkey, ballot) 
		values (?1, ?2, ?4, ?5);`,
		func(stmt *sql.Statement) {
			stmt.BindBytes(1, ballot.ID().Bytes())
			stmt.BindInt64(2, int64(ballot.Layer.Value))
			stmt.BindBytes(4, ballot.SmesherID().Bytes())
			stmt.BindBytes(5, bytes)
		}, nil); err != nil {
		return fmt.Errorf("insert ballot %s: %w", ballot.ID(), err)
	}
	return nil
}

// Has a ballot in the database.
func Has(db sql.Executor, id types.BallotID) (bool, error) {
	rows, err := db.Exec("select 1 from ballots where id = ?1;",
		func(stmt *sql.Statement) {
			stmt.BindBytes(1, id.Bytes())
		}, nil,
	)
	if err != nil {
		return false, fmt.Errorf("has ballot %s: %w", id, err)
	}
	return rows > 0, nil
}

// Get ballot with id from database.
func Get(db sql.Executor, id types.BallotID) (rst *types.Ballot, err error) {
	if rows, err := db.Exec(`select pubkey, ballot, length(identities.proof)
	from ballots left join identities using(pubkey)
	where id = ?1;`,
		func(stmt *sql.Statement) {
			stmt.BindBytes(1, id.Bytes())
		}, func(stmt *sql.Statement) bool {
			rst, err = decodeBallot(id,
				stmt.ColumnReader(0),
				stmt.ColumnReader(1),
				stmt.ColumnInt(2) > 0,
			)
			return true
		}); err != nil {
		return nil, fmt.Errorf("get %s: %w", id, err)
	} else if rows == 0 {
		return nil, fmt.Errorf("%w ballot %s", sql.ErrNotFound, id)
	}
	return rst, nil
}

// Layer returns full body ballot for layer.
func Layer(db sql.Executor, lid types.LayerID) (rst []*types.Ballot, err error) {
	if _, err = db.Exec(`select id, pubkey, ballot, length(identities.proof)
		from ballots left join identities using(pubkey)
		where layer = ?1;`, func(stmt *sql.Statement) {
		stmt.BindInt64(1, int64(lid.Value))
	}, func(stmt *sql.Statement) bool {
		id := types.BallotID{}
		stmt.ColumnBytes(0, id[:])
		var ballot *types.Ballot
		ballot, err = decodeBallot(id,
			stmt.ColumnReader(1),
			stmt.ColumnReader(2),
			stmt.ColumnInt(3) > 0,
		)
		if err != nil {
			return false
		}
		rst = append(rst, ballot)
		return true
	}); err != nil {
		return nil, fmt.Errorf("ballots for layer %s: %w", lid, err)
	}
	return rst, err
}

// IDsInLayer returns ballots ids in the layer.
func IDsInLayer(db sql.Executor, lid types.LayerID) (rst []types.BallotID, err error) {
	if _, err := db.Exec("select id from ballots where layer = ?1;", func(stmt *sql.Statement) {
		stmt.BindInt64(1, int64(lid.Uint32()))
	}, func(stmt *sql.Statement) bool {
		id := types.BallotID{}
		stmt.ColumnBytes(0, id[:])
		rst = append(rst, id)
		return true
	}); err != nil {
		return nil, fmt.Errorf("ballots for layer %s: %w", lid, err)
	}
	return rst, err
}

// CountByPubkeyLayer counts number of ballots in the layer for the pubkey.
func CountByPubkeyLayer(db sql.Executor, lid types.LayerID, pubkey []byte) (int, error) {
	rows, err := db.Exec("select 1 from ballots where layer = ?1 and pubkey = ?2;", func(stmt *sql.Statement) {
		stmt.BindInt64(1, int64(lid.Value))
		stmt.BindBytes(2, pubkey)
	}, nil)
	if err != nil {
		return 0, fmt.Errorf("counting layer %s: %w", lid, err)
	}
	return rows, nil
}

// LayerBallotByNodeID returns any ballot by the specified NodeID in a given layer.
func LayerBallotByNodeID(db sql.Executor, lid types.LayerID, nodeID types.NodeID) (*types.Ballot, error) {
	var (
		ballot types.Ballot
		err    error
	)
	enc := func(stmt *sql.Statement) {
		stmt.BindInt64(1, int64(lid.Value))
		stmt.BindBytes(2, nodeID.Bytes())
	}
	dec := func(stmt *sql.Statement) bool {
		var (
			n   int
			bid types.BallotID
		)
		stmt.ColumnBytes(0, bid[:])
		if n, err = codec.DecodeFrom(stmt.ColumnReader(1), &ballot); err != nil {
			if err != io.EOF {
				err = fmt.Errorf("ballot data layer %v, nodeID %v: %w", lid, nodeID, err)
				return false
			}
		} else if n == 0 {
			err = fmt.Errorf("ballot data missing layer %v, nodeID %v", lid, nodeID)
			return false
		}
		ballot.SetID(bid)
		ballot.SetSmesherID(nodeID)
		return true
	}
	if rows, err := db.Exec(`
		select id, ballot from ballots
		where layer = ?1 and pubkey = ?2
		limit 1;`, enc, dec); err != nil {
		return nil, fmt.Errorf("same layer ballot %v: %w", lid, err)
	} else if rows == 0 {
		return nil, sql.ErrNotFound
	}
	return &ballot, err
}

// GetRefBallot gets a ref ballot for a layer and a pubkey.
func GetRefBallot(db sql.Executor, epochID types.EpochID, pubkey []byte) (ballotID types.BallotID, err error) {
	firstLayer := epochID.FirstLayer()
	lastLayer := firstLayer.Add(types.GetLayersPerEpoch()).Sub(1)
	rows, err := db.Exec(`
		select id from ballots 
		where layer between ?1 and ?2 and pubkey = ?3
		order by layer
		limit 1;`,
		func(stmt *sql.Statement) {
			stmt.BindInt64(1, int64(firstLayer.Value))
			stmt.BindInt64(2, int64(lastLayer.Value))
			stmt.BindBytes(3, pubkey)
		}, func(stmt *sql.Statement) bool {
			stmt.ColumnBytes(0, ballotID[:])
			return true
		})
	if err != nil {
		return types.BallotID{}, fmt.Errorf("ref ballot epoch %v: %w", epochID, err)
	} else if rows == 0 {
		return types.BallotID{}, fmt.Errorf("%w ref ballot epoch %s", sql.ErrNotFound, epochID)
	}
	return ballotID, nil
}

// LatestLayer gets the highest layer with ballots.
func LatestLayer(db sql.Executor) (types.LayerID, error) {
	var lid types.LayerID
	if _, err := db.Exec("select max(layer) from ballots;",
		nil,
		func(stmt *sql.Statement) bool {
			lid = types.NewLayerID(uint32(stmt.ColumnInt64(0)))
			return true
		}); err != nil {
		return lid, fmt.Errorf("latest layer: %w", err)
	}
	return lid, nil
}
