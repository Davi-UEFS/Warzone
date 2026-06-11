package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"time"

	"github.com/Davi-UEFS/Warzone/shared"
	"github.com/Davi-UEFS/Warzone/warzone-core/x/warzone/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cosmos/gogoproto/grpc"

	"google.golang.org/api/transport/grpc"
	"google.golang.org/grpc"
)

type Wallet struct{
	Name string
	Addr string
	PrivKey secp256k1.PrivKey
	PubKey secp256k1.PubKey
}

func (w *Wallet) CreateWallet(){

	w.PrivKey = secp256k1.GenPrivKey()

	w.PubKey = w.PrivKey.PubKey()

	w.Addr = sdk.AccAddress(w.PubKey.Address())

}

func ConnectBlockchain(addr string) (types.QueryClient, error){

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := types.NewQueryClient(conn)
	return client, nil

}

func StartBlockchainNode() error {
    cmd := exec.Command(
        "warzoned",
        "start",
    )

    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    return cmd.Start()


}


func RegisterDrone (d shared.Drone, w Wallet){

	msg := &types.MsgRegDrone{
		Creator: w.Addr,
		DroneId: d.ID,
		Sector: d.CurrentSector,
		Battery: d.BatteryLevel,
	}

	txBuilder := txConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)

	err = tx.Sign(
		txConfig,
		"empresa",
		txBuilder,
		signerData,
		privKey,
		true,
	)

	txBytes, err := txConfig.TxEncoder()(
    txBuilder.GetTx(),
)

	res, err := txClient.BroadcastTx(
		context.Background(),
		&txtypes.BroadcastTxRequest{
			Mode: txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
			TxBytes: txBytes,
		},
	)
	
}

func ReportDeadDrone ()

// FetchMissionsPENDING simulado para o laboratório (não precisa de blockchain rodando)
func FetchMissionsPENDING(blockchainURL string, targetSector string) ([]types.Mission, error) {
	fmt.Println("[MOCK] Simulando leitura de dados da Blockchain...")

	// Simulamos o JSON que a blockchain enviaria se estivesse ligada
	mockMissions := []types.Mission{
		{
			Id:              1,
			Creator:         "cosmos1xyz...",
			Sector:          targetSector,
			Status:          "PENDING",
			Priority:        2, // Alta
			AssignedDroneId: "",
			ReqType:         "THROW_WATER",
			Coord:           "12.34,-56.78",
			CreatedAt:       1717941234,
		},
		{
			Id:              2,
			Creator:         "cosmos1abc...",
			Sector:          targetSector,
			Status:          "PENDING",
			Priority:        1, // Baixa
			AssignedDroneId: "",
			ReqType:         "INSPECT_AREA",
			Coord:           "12.99,-56.11",
			CreatedAt:       1717941500,
		},
	}

	return mockMissions, nil
}

func FetchDronesIDLE(blockchainURL string, targetSector string) ([]types.Drone, error) {
	fmt.Println("[MOCK] Simulando leitura de dados da Blockchain...")

	// Simulamos o JSON que a blockchain enviaria se estivesse ligada
	mockDrones := []types.Drone{
		{
			DroneId: "drone-1",
			Sector:  targetSector,
			Status:  "IDLE",
			Battery: "85", //TODO: ajustar tipo de dado

		},
	}

	return mockDrones, nil
}

func sortRequisitionsWithAging(requisitions []types.Mission) {

	if len(requisitions) == 0 {
		return
	}

	//TODO: implementar ordenação por prioridade e tempo de espera (aging) para evitar starvation

	/*
		sort.Slice(requisitions, func(i, j int) bool {
			now := time.Now().Unix()

			// 1. Converter os timestamps da blockchain (string) para int64
			timeI, _ := requisitions[i].CreatedAt
			timeJ, _ :=

			waitTimeI := now - timeI
			PrioBoostI := waitTimeI / 10
			scoreI := PrioBoostI + waitTimeI

			waitTimeJ := now - timeJ
			PrioBoostJ := waitTimeJ / 10
			scoreJ := PrioBoostJ + waitTimeJ

			// Ordena pelo maior score final
			if scoreI != scoreJ {
				return scoreI > scoreJ
			}
			// Desempate pelo ID menor
			return requisitions[i].Id < requisitions[j].Id
		})
	*/
}

func newDispatch() {

	requisitions, err := FetchMissionsPENDING("http://temp-blockchain-url", "Sector-1")
	if err != nil {
		fmt.Println("Erro ao buscar missões:", err)
		return
	}

	// TODO: FAZER LOGICA DE DESPACHO AQUI
	print(requisitions)

}
