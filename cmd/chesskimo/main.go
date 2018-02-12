package main

import (
	"fmt"
	"sort"

	"github.com/dbriemann/chesskimo"
)

var version = "undefined"

func main() {
	fmt.Println("Version", version)

	fens := []string{
		// DIVIDE TESTS (http://www.rocechess.ch/perft.html)
		//		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", // 1. Start position -- tested and validated until depth 7

		//		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", // 2. Good testposition -- tested and validated until depth 6

		//		"n1n5/PPPk4/8/8/8/8/4Kppp/5N1N b - - 0 1", // 3. Discover promotion bugs -- tested and validated until depth 6

		//		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 0", // Kiwipete -- tested and validated until depth 5

		//		"r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1", // Dense -- tested and validated until depth 5

		"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 0", // Endgame

	}

	b := chesskimo.NewBoard()
	//	mlist := base.MoveList{}

	for _, fen := range fens {
		b.SetFEN(fen)
		fmt.Println(&b)

		result := b.PerftDivide(5)
		total := uint64(0)

		keys := make([]string, 0, len(result))
		for key := range result {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Println(key, result[key])
			total += result[key]
		}
		fmt.Println("TOTAL", total)

		//		for i := 1; i <= 5; i++ {
		//			//			start := time.Now()
		//			//			moves, captures, checks := b.AnalyzerPerft(i, 0, 0)
		//			//			end := time.Now()
		//			//			fmt.Printf("Perft %d : %d moves (%d captures, %d checks) in %f seconds.\n", i, moves, captures, checks, end.Sub(start).Seconds())

		//			start := time.Now()
		//			moves := b.FastPerft(i)
		//			end := time.Now()
		//			fmt.Printf("Perft %d : %d moves in %f seconds.\n", i, moves, end.Sub(start).Seconds())
		//		}

		//		mlist.Clear()
		//		b.GenerateAllLegalMoves(&mlist, b.Player)

		//		//		b.GenerateKingMoves(&mlist, b.Player)
		//		//		b.GenerateQueenMoves(&mlist, b.Player)
		//		//		b.GenerateRookMoves(&mlist, b.Player)
		//		//		b.GenerateBishopMoves(&mlist, b.Player)
		//		//		b.GenerateKnightMoves(&mlist, b.Player)
		//		//		b.GeneratePawnMoves(&mlist, b.Player)

		//		fmt.Println("Moves:")
		//		fmt.Println(&mlist)

		//		b.MakeLegalMove(mlist.Get(4)) // 0-0-0
		//		fmt.Println(&b)
		//		mlist.Clear()
		//		b.GenerateAllLegalMoves(&mlist, b.Player)
		//		fmt.Println("Moves:")
		//		fmt.Println(&mlist)
	}

}
