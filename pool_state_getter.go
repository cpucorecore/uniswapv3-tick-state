package main

import "github.com/ethereum/go-ethereum/common"

func GetPoolStateFromDBOrContractCaller(db Repo, cc *ContractCaller, addr common.Address) (*PoolState, error) {
	ok, err := db.PoolExists(addr)
	if err != nil || !ok {
		poolState, err := cc.GetPoolState(addr)
		if err != nil {
			return nil, err
		}

		err = db.SetPoolState(addr, poolState)
		if err != nil {
			return nil, err
		}

		return poolState, nil
	}

	return db.GetPoolState(addr)
}
