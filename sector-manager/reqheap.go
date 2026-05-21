package main

import (
	"container/heap"
	"fmt"

	"github.com/Davi-UEFS/Warzone/shared"
)

// ReqHeap é uma fila de prioridade que usa heap (árvore binária) para armazenar as requisições pendentes.
// Ele é ordenada primeiro pela prioridade (maior prioridade primeiro) e depois pelo tempo de Lamport.
type ReqHeap []shared.Requisition

func (h ReqHeap) Len() int { return len(h) }

// Less é a implementação da função de comparação para o heap. Ela ordena as requisições primeiro pela prioridade
// (maior primeiro) e, em caso de empate, pelo tempo de Lamport (menor primeiro).
func (h ReqHeap) Less(i, j int) bool {
	if h[i].Priority != h[j].Priority {
		return h[i].Priority > h[j].Priority
	}
	return h[i].LamportTime < h[j].LamportTime
}

// Swap troca os elementos nas posições i e j.
func (h ReqHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

// Push adiciona um elemento ao heap. Ele é chamado pelo pacote heap quando um novo elemento é adicionado.
func (h *ReqHeap) Push(x interface{}) {
	*h = append(*h, x.(shared.Requisition))
}

// Pop remove e retorna o elemento de maior prioridade do heap. Ele é chamado pelo pacote heap quando um elemento é removido.
func (h *ReqHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// Peek retorna o elemento de maior prioridade do heap sem removê-lo. Ele é útil para verificar qual é a próxima requisição a ser processada sem modificar o heap.
func (h *ReqHeap) Peek() shared.Requisition {
	return (*h)[0]
}

// ToSlice converte o heap em um slice de requisições. Ele é útil para serializar o estado do heap ou para realizar operações que exigem um slice,
// como na hora de fazer o snapshot da FSM.
func (h *ReqHeap) ToSlice() []shared.Requisition {
	out := make([]shared.Requisition, len(*h))
	copy(out, *h)
	return out
}

// FromSlice atualiza o heap a partir de um slice de requisições. Ele é útil para restaurar o estado do heap a partir de um snapshot da FSM.
func (h *ReqHeap) FromSlice(s []shared.Requisition) {
	*h = make([]shared.Requisition, len(s))
	copy(*h, s)
	heap.Init(h)
}

// RemoveAt remove a requisição na posição i do heap.
func (h *ReqHeap) RemoveAt(i int) shared.Requisition {
	x := heap.Remove(h, i).(shared.Requisition)
	return x
}

// ApplyAging aplica o mecanismo de aging para evitar starvation. Ele percorre todas as requisições no heap e verifica se alguma delas
// ultrapassou o tempo limite definido. Se uma requisição ultrapassou o tempo limite, ela aumenta sua prioridade.
//
// Após aplicar o aging, o heap é reordenado para refletir as mudanças de prioridade.
//
// Params:
//   - currentTime: o tempo atual em segundos (geralmente obtido com time.Now().Unix()).
//   - thresholdSeconds: o tempo limite em segundos.
//   - boostAmount: a quantidade de prioridade a ser adicionada às requisições que ultrapassaram o tempo limite.
func (h *ReqHeap) ApplyAging(currentTime int64, thresholdSeconds int64, boostAmount int) {
	for i := range *h {
		age := currentTime - (*h)[i].CreatedAt
		if age > thresholdSeconds {
			oldPriority := (*h)[i].Priority
			(*h)[i].Priority += boostAmount

			// ----------------------------------------------------
			// Modo de debug para mostrar o aging em ação.
			// Esta seção não é necessária para o funcionamento normal do sistema..
			// ----------------------------------------------------
			if DebugMode {
				fmt.Printf("\n\033[1;35m[DEBUG-AGING]\033[0m Missão %s envelheceu (%ds). Boost: %d -> %d\n",
					(*h)[i].ID, age, oldPriority, (*h)[i].Priority)
			}
		}
	}
	heap.Init(h)
}
