package chesskimo

import (
	"fmt"
	"strconv"
)

// Board contains all information for a chess board state, including the board itself as 0x88 board.
//
//   +------------------------+
// 8 |70 71 72 73 74 75 76 77 | 78 79 7a 7b 7c 7d 7e 7f
// 7 |60 61 62 63 64 65 66 67 | 68 69 6a 6b 6c 6d 6e 6f
// 6 |50 51 52 53 54 55 56 57 | 58 59 5a 5b 5c 5d 5e 5f
// 5 |40 41 42 43 44 45 46 47 | 48 49 4a 4b 4c 4d 4e 4f
// 4 |30 31 32 33 34 35 36 37 | 38 39 3a 3b 3c 3d 3e 3f
// 3 |20 21 22 23 24 25 26 27 | 28 29 2a 2b 2c 2d 2e 2f
// 2 |10 11 12 13 14 15 16 17 | 18 19 1a 1b 1c 1d 1e 1f
// 1 | 0  1  2  3  4  5  6  7 |  8  9  a  b  c  d  e  f
//   +------------------------+
//     a  b  c  d  e  f  g  h
//
// All indexes shown are in HEX. The left board represents the actual playing board, whereas there right board
// is for detection of illegal moves without heavy conditional usage.
type Board struct {
	Squares     [64 * 2]Piece
	CastleShort [2]bool
	CastleLong  [2]bool
	MoveNumber  uint16
	DrawCounter uint16
	CheckInfo   Square
	EpSquare    Square
	Player      Color
	Kings       [2]Square
	Sliders     [2]PieceList
	Queens      [2]PieceList
	Rooks       [2]PieceList
	Bishops     [2]PieceList
	Knights     [2]PieceList
	Pawns       [2]PieceList
}

const (
	CHECK_NONE         = OTB
	CHECK_DOUBLE_CHECK = 0x0F // Is also off the board but used as double check value.
	CHECK_CHECKMATE    = 0x1F
)

var (
	// Lookup0x88 maps the indexes of a 8x8 MinBoard to the 0x88 board indexes.
	Lookup0x88 = [64]Square{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
		0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37,
		0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47,
		0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57,
		0x60, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67,
		0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77,
	}
	SQUARE_DIFFS = [240]Piece{}
	DIFF_DIRS    = [240]int8{}
)

func init() {
	populateDiffMaps()
}

func populateDiffMaps() {
	diffdir := int8(0)
	for _, from := range Lookup0x88 {
		for _, to := range Lookup0x88 {
			diff := uint8(0x77 + (int8(from) - int8(to)))
			pbits := Piece(0) // Resulting bitset.

			// Test if diff is a possible move for all types of pieces.
			// kings:
			for _, dir := range KING_DIRS {
				target := Square(int8(from) + dir)

				if target == to {
					pbits |= KING
					break
				}
			}

			// diagonal sliders:
			for _, dir := range DIAGONAL_DIRS {
				for steps := int8(1); ; steps++ {
					target := Square(int8(from) + steps*dir)
					target.Flip()
					if !target.OnBoard() {
						// We left the board -> next dir.
						break
					}

					if target == to {
						diffdir = dir
						// Can be reachedby queens and bishops (diagonally).
						pbits |= BISHOP | QUEEN
						// The goto acts as a double break.
						// Skipping orthos and knights.
						goto INSPECTION_FINISHED
					}
				}
			}

			// orthogonal sliders:
			for _, dir := range ORTHOGONAL_DIRS {
				for steps := int8(1); ; steps++ {
					target := Square(int8(from) + steps*dir)
					if !target.OnBoard() {
						// We left the board -> next dir.
						break
					}

					if target == to {
						diffdir = dir
						// Can be reachedby queens and bishops (diagonally).
						pbits |= ROOK | QUEEN
						// The goto acts as a double break.
						// Skipping knights.
						goto INSPECTION_FINISHED
					}
				}
			}

			// knights:
			for _, dir := range KNIGHT_DIRS {
				target := Square(int8(from) + dir)

				if target == to {
					diffdir = dir
					pbits |= KNIGHT
					break
				}
			}

		INSPECTION_FINISHED:

			DIFF_DIRS[diff] = diffdir
			SQUARE_DIFFS[diff] = pbits
		}
	}
}

func NewBoard() Board {
	b := Board{}
	b.Kings = [2]Square{OTB, OTB}
	b.Sliders = [2]PieceList{NewPieceList()}
	b.Queens = [2]PieceList{NewPieceList()}
	b.Rooks = [2]PieceList{NewPieceList()}
	b.Bishops = [2]PieceList{NewPieceList()}
	b.Knights = [2]PieceList{NewPieceList()}
	b.Pawns = [2]PieceList{NewPieceList()}

	b.CheckInfo = CHECK_NONE
	b.SetStartingPosition()
	return b
}

// SetStartingPosition sets the starting position of a default chess game via FEN code.
func (b *Board) SetStartingPosition() {
	err := b.SetFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		panic(err)
	}
}

// SetFEN sets a chess position from a 'FEN' string. This function is expensive
// in terms of computation time. It should only be used in non performance critical
// code such as setting up states etc.
func (b *Board) SetFEN(fenstr string) error {
	mb, err := ParseFEN(fenstr)
	if err != nil {
		return err
	}

	for color := BLACK; color <= WHITE; color++ {
		b.Sliders[color].Clear()
		b.Queens[color].Clear()
		b.Rooks[color].Clear()
		b.Bishops[color].Clear()
		b.Knights[color].Clear()
		b.Pawns[color].Clear()
	}

	// No error encountered -> set all members of the board instance.
	b.MoveNumber = mb.MoveNum
	b.DrawCounter = mb.HalfMoves

	if mb.EpSquare != OTB {
		b.EpSquare = Lookup0x88[mb.EpSquare]
	} else {
		b.EpSquare = OTB
	}

	b.CastleShort = mb.CastleShort
	b.CastleLong = mb.CastleLong
	b.Player = mb.Color
	for idx, piece := range mb.Squares {
		sq := Lookup0x88[idx]
		b.Squares[sq] = piece

		switch piece {
		case WKING:
			b.Kings[WHITE] = sq
		case BKING:
			b.Kings[BLACK] = sq
		case WQUEEN:
			b.Queens[WHITE].Add(sq)
			b.Sliders[WHITE].Add(sq)
		case BQUEEN:
			b.Queens[BLACK].Add(sq)
			b.Sliders[BLACK].Add(sq)
		case WROOK:
			b.Rooks[WHITE].Add(sq)
			b.Sliders[WHITE].Add(sq)
		case BROOK:
			b.Rooks[BLACK].Add(sq)
			b.Sliders[BLACK].Add(sq)
		case WBISHOP:
			b.Bishops[WHITE].Add(sq)
			b.Sliders[WHITE].Add(sq)
		case BBISHOP:
			b.Bishops[BLACK].Add(sq)
			b.Sliders[BLACK].Add(sq)
		case WKNIGHT:
			b.Knights[WHITE].Add(sq)
		case BKNIGHT:
			b.Knights[BLACK].Add(sq)
		case WPAWN:
			b.Pawns[WHITE].Add(sq)
		case BPAWN:
			b.Pawns[BLACK].Add(sq)
		}
	}

	// Set info board and find possible checks.
	b.DetectChecksAndPins(b.Player)

	return nil
}

func (b *Board) clearMetaInfo() {
	b.CheckInfo = CHECK_NONE
	// Manually unrolled loop. Ugly but MUCH faster than looping over the index lookup
	// or counting up the right side board indexes with (i + 1) | 0x8.
	b.Squares[0x8] = INFO_NONE
	b.Squares[0x9] = INFO_NONE
	b.Squares[0xa] = INFO_NONE
	b.Squares[0xb] = INFO_NONE
	b.Squares[0xc] = INFO_NONE
	b.Squares[0xd] = INFO_NONE
	b.Squares[0xe] = INFO_NONE
	b.Squares[0xf] = INFO_NONE
	b.Squares[0x18] = INFO_NONE
	b.Squares[0x19] = INFO_NONE
	b.Squares[0x1a] = INFO_NONE
	b.Squares[0x1b] = INFO_NONE
	b.Squares[0x1c] = INFO_NONE
	b.Squares[0x1d] = INFO_NONE
	b.Squares[0x1e] = INFO_NONE
	b.Squares[0x1f] = INFO_NONE
	b.Squares[0x28] = INFO_NONE
	b.Squares[0x29] = INFO_NONE
	b.Squares[0x2a] = INFO_NONE
	b.Squares[0x2b] = INFO_NONE
	b.Squares[0x2c] = INFO_NONE
	b.Squares[0x2d] = INFO_NONE
	b.Squares[0x2e] = INFO_NONE
	b.Squares[0x2f] = INFO_NONE
	b.Squares[0x38] = INFO_NONE
	b.Squares[0x39] = INFO_NONE
	b.Squares[0x3a] = INFO_NONE
	b.Squares[0x3b] = INFO_NONE
	b.Squares[0x3c] = INFO_NONE
	b.Squares[0x3d] = INFO_NONE
	b.Squares[0x3e] = INFO_NONE
	b.Squares[0x3f] = INFO_NONE
	b.Squares[0x48] = INFO_NONE
	b.Squares[0x49] = INFO_NONE
	b.Squares[0x4a] = INFO_NONE
	b.Squares[0x4b] = INFO_NONE
	b.Squares[0x4c] = INFO_NONE
	b.Squares[0x4d] = INFO_NONE
	b.Squares[0x4e] = INFO_NONE
	b.Squares[0x4f] = INFO_NONE
	b.Squares[0x58] = INFO_NONE
	b.Squares[0x59] = INFO_NONE
	b.Squares[0x5a] = INFO_NONE
	b.Squares[0x5b] = INFO_NONE
	b.Squares[0x5c] = INFO_NONE
	b.Squares[0x5d] = INFO_NONE
	b.Squares[0x5e] = INFO_NONE
	b.Squares[0x5f] = INFO_NONE
	b.Squares[0x68] = INFO_NONE
	b.Squares[0x69] = INFO_NONE
	b.Squares[0x6a] = INFO_NONE
	b.Squares[0x6b] = INFO_NONE
	b.Squares[0x6c] = INFO_NONE
	b.Squares[0x6d] = INFO_NONE
	b.Squares[0x6e] = INFO_NONE
	b.Squares[0x6f] = INFO_NONE
	b.Squares[0x78] = INFO_NONE
	b.Squares[0x79] = INFO_NONE
	b.Squares[0x7a] = INFO_NONE
	b.Squares[0x7b] = INFO_NONE
	b.Squares[0x7c] = INFO_NONE
	b.Squares[0x7d] = INFO_NONE
	b.Squares[0x7e] = INFO_NONE
	b.Squares[0x7f] = INFO_NONE
}

//func (b *Board) Result(mlist *MoveList) State {

//}

// MakeLegalMove expects a legal move and applies it to the board.
func (b *Board) MakeLegalMove(m BitMove) {
	from, to, promo := m.All()
	oppColor := b.Player.Flip()
	// Detect piece type and target piece
	ptype := b.Squares[from] & PIECE_MASK
	tpiece := b.Squares[to]

	// Test if it is a capture.
	if !tpiece.IsEmpty() {
		if tpiece.Contains(KING) {
			panic("\n" + b.InfoBoardString() + "\n" + b.String() + "\nfor move:" + m.MiniNotation() + "\n")
		}
		// Remove captured piece from the board.
		b.removePiece(to)

		// If capture captures a rook disable that side for castling.
		// TODO this could be realized differently:
		// check castling squares for specific values.. add one for rook.
		if to == CASTLING_ROOK_SHORT[oppColor] {
			b.CastleShort[oppColor] = false
		} else if to == CASTLING_ROOK_LONG[oppColor] {
			b.CastleLong[oppColor] = false
		}
	} else if ptype == PAWN && to == b.EpSquare { // Is it an e.p. capture?
		capSq := Square(int8(to) + PAWN_PUSH_DIRS[oppColor])
		b.removePiece(capSq)
	}
	// Now make the actual move on the board.
	b.Squares[to], b.Squares[from] = b.Squares[from], EMPTY
	// Remove any possible e.p. squares.
	b.EpSquare = OTB

	// Make move in piece list and do special move things.
	switch ptype {
	case PAWN:
		if from.CreatesEnPassent(to) {
			b.EpSquare = Square(int8(from) + PAWN_PUSH_DIRS[b.Player])
		}
		if promo != NONE {
			b.Pawns[b.Player].Remove(from)
			b.addPiece(to, promo|b.Player)
		} else {
			b.Pawns[b.Player].Move(from, to)
		}
	case KNIGHT:
		b.Knights[b.Player].Move(from, to)
	case BISHOP:
		b.Bishops[b.Player].Move(from, to)
		b.Sliders[b.Player].Move(from, to)
	case ROOK:
		if from == CASTLING_ROOK_SHORT[b.Player] {
			b.CastleShort[b.Player] = false
		} else if from == CASTLING_ROOK_LONG[b.Player] {
			b.CastleLong[b.Player] = false
		}
		b.Rooks[b.Player].Move(from, to)
		b.Sliders[b.Player].Move(from, to)
	case QUEEN:
		b.Queens[b.Player].Move(from, to)
		b.Sliders[b.Player].Move(from, to)
	case KING:
		b.Kings[b.Player] = to
		b.CastleShort[b.Player] = false
		b.CastleLong[b.Player] = false

		shortCastle := (from == CASTLING_DETECT_SHORT[b.Player][0]) && (to == CASTLING_DETECT_SHORT[b.Player][1])
		longCastle := (from == CASTLING_DETECT_LONG[b.Player][0]) && (to == CASTLING_DETECT_LONG[b.Player][1])
		// Let's teleport the rook, if the move is castling.
		if shortCastle {
			rookFrom := CASTLING_ROOK_SHORT[b.Player]
			rookTo := CASTLING_PATH_SHORT[b.Player][0]
			b.Squares[rookTo], b.Squares[rookFrom] = ROOK|b.Player, EMPTY
			b.Rooks[b.Player].Move(rookFrom, rookTo)
			b.Sliders[b.Player].Move(rookFrom, rookTo)
		} else if longCastle {
			rookFrom := CASTLING_ROOK_LONG[b.Player]
			rookTo := CASTLING_PATH_LONG[b.Player][0]
			b.Squares[rookTo], b.Squares[rookFrom] = ROOK|b.Player, EMPTY
			b.Rooks[b.Player].Move(rookFrom, rookTo)
			b.Sliders[b.Player].Move(rookFrom, rookTo)
		}

	default:
		// This should not happen..
		panic("Board.MakeLegalMove: " + fmt.Sprintf("%v", m))
	}

	b.Player = b.Player.Flip()
	b.MoveNumber++
	// TODO half draw counter
}

// TODO (improvement) -> introduce movePiece function..

func (b *Board) addPiece(sq Square, piece Piece) {
	b.Squares[sq] = piece
	ptype := piece & PIECE_MASK
	color := piece.PieceColor()

	switch ptype {
	case PAWN:
		b.Pawns[color].Add(sq)
	case KNIGHT:
		b.Knights[color].Add(sq)
	case BISHOP:
		b.Bishops[color].Add(sq)
		b.Sliders[color].Add(sq)
	case ROOK:
		b.Rooks[color].Add(sq)
		b.Sliders[color].Add(sq)
	case QUEEN:
		b.Queens[color].Add(sq)
		b.Sliders[color].Add(sq)
	default:
		// This should not happen..
		panic("Board.addPiece: " + fmt.Sprintf("%s %s", PrintBoardIndex[sq], PrintMap[piece]))
	}
}

func (b *Board) removePiece(sq Square) {
	piece := b.Squares[sq]
	ptype := piece & PIECE_MASK
	color := piece.PieceColor()

	b.Squares[sq] = EMPTY
	switch ptype {
	case PAWN:
		b.Pawns[color].Remove(sq)
	case KNIGHT:
		b.Knights[color].Remove(sq)
	case BISHOP:
		b.Bishops[color].Remove(sq)
		b.Sliders[color].Remove(sq)
	case ROOK:
		b.Rooks[color].Remove(sq)
		b.Sliders[color].Remove(sq)
	case QUEEN:
		b.Queens[color].Remove(sq)
		b.Sliders[color].Remove(sq)
	default:
		// This should not happen..
		panic("Board.removePiece: " + fmt.Sprintf("%s %s", PrintBoardIndex[sq], PrintMap[piece]))
	}
}

// IsSquareAttacked tests if a specific square is attacked by any enemy.
func (b *Board) IsSquareAttacked(sq, ignoreSq Square, color Color) bool {
	oppColor := color.Flip()

	// 1. Detect attacks	by knights.
	for i := uint8(0); i < b.Knights[oppColor].Size; i++ {
		knightSq := b.Knights[oppColor].Pieces[i]
		if knightSq == ignoreSq {
			continue
		}
		diff := knightSq.Diff(sq)
		if SQUARE_DIFFS[diff].Contains(KNIGHT) {
			return true
		}
	}

	// 3. Detect attacks by sliders.
	if b.IsSqAttackedBySlider(color, sq, ignoreSq) {
		return true
	}
	//	if b.IsSqAttackedBySlider(color, sq, ignoreSq, &b.Queens[oppColor], QUEEN) {
	//		return true
	//	}
	//	if b.IsSqAttackedBySlider(color, sq, ignoreSq, &b.Bishops[oppColor], BISHOP) {
	//		return true
	//	}
	//	if b.IsSqAttackedBySlider(color, sq, ignoreSq, &b.Rooks[oppColor], ROOK) {
	//		return true
	//	}

	// 2. Detect attacks by pawns.
	oppPawn := PAWN | oppColor

	maybePawnSq := Square(int8(sq) + PAWN_CAPTURE_DIRS[color][0])
	if maybePawnSq.OnBoard() && b.Squares[maybePawnSq] == oppPawn {
		// Found an attacking pawn by inspecting in reverse direction.
		return true
	}
	maybePawnSq = Square(int8(sq) + PAWN_CAPTURE_DIRS[color][1])
	if maybePawnSq.OnBoard() && b.Squares[maybePawnSq] == oppPawn {
		// Found an attacking pawn by inspecting in reverse direction.
		return true
	}

	// 0. Detect attacks by kings.
	oppKingSq := b.Kings[oppColor]
	diff := oppKingSq.Diff(sq)
	if SQUARE_DIFFS[diff].Contains(KING) {
		return true
	}

	return false
}

// IsSqAttackedBySlider tests if a specific square is attacked by an enemy slider.
func (b *Board) IsSqAttackedBySlider(color Color, sq, ignoreSq Square) bool {
	oppColor := color.Flip()
	for i := uint8(0); i < b.Sliders[oppColor].Size; i++ {
		sliderSq := b.Sliders[oppColor].Pieces[i]
		ptype := b.Squares[sliderSq] & PIECE_MASK
		if sliderSq == ignoreSq {
			// This only evals to true in pawn move legality tests.
			// If the pawn captured a piece this piece is still in the piece list
			// and has to be ignored here. (It also works without this but
			// this way we can save some checks.. TODO: test).
			continue
		}
		diff := sq.Diff(sliderSq)
		if SQUARE_DIFFS[diff].Contains(ptype) {
			// The slider possibly attacks sq.
			diffdir := DIFF_DIRS[diff]
			// Starting from sq we step through the path in question towards the enemy slider.
			for stepSq := Square(int8(sq) + diffdir); ; stepSq = Square(int8(stepSq) + diffdir) {
				curPiece := b.Squares[stepSq]
				if curPiece.IsEmpty() {
					continue
				} else if curPiece.HasColor(color) {
					// A friendly piece was encountered -> no attack -> next piece.
					break
				} else if curPiece.Contains(ptype) {
					// An attacking enemy was encountered.
					return true
				} else {
					// A blocking enemy piece was encountered -> no attack -> next piece.
					break
				}
			}
		}
	}

	return false
}

func (b *Board) DetectChecksAndPins(color Color) {
	b.clearMetaInfo()
	oppColor := color.Flip()

	kingSq := b.Kings[color]
	checkCounter := 0
	pinMarker := INFO_PIN_FIRST

	// 1. Detect checks by knights.
	for i := uint8(0); i < b.Knights[oppColor].Size; i++ {
		knightSq := b.Knights[oppColor].Pieces[i]
		diff := knightSq.Diff(kingSq)
		if SQUARE_DIFFS[diff].Contains(KNIGHT) {
			// The knight checks the king. Save the info and break the loop.
			// There can never be a double check of 2 knights.
			b.CheckInfo = knightSq
			checkCounter++
			b.Squares[knightSq.ToInfoIndex()].Set(INFO_MASK_CHECK)
			break
		}
	}
	//	for i := uint8(0); i < b.Knights[oppColor].Size; i++ {
	//		from := b.Knights[oppColor].Pieces[i]
	//		for d := 0; d < 8; d++ {
	//			dir := KNIGHT_DIRS[d]
	//			to := Square(int8(from) + dir)
	//			if to.OnBoard() {
	//				if to == kingSq {
	//					b.CheckInfo = to
	//					checkCounter++
	//					b.Squares[to.ToInfoIndex()] = INFO_CHECK
	//				} else {
	//					b.Squares[to.ToInfoIndex()] = INFO_ATTACKED
	//				}
	//			}
	//		}
	//	}

	// 2. Detect checks by pawns.
	for i := uint8(0); i < b.Pawns[oppColor].Size; i++ {
		pawnSq := b.Pawns[oppColor].Pieces[i]
		//		for _, dir := range PAWN_CAPTURE_DIRS[oppColor] {
		for d := 0; d < 2; d++ {
			dir := PAWN_CAPTURE_DIRS[oppColor][d]
			to := Square(int8(pawnSq) + dir)
			if kingSq == to {
				// The pawn checks the king. Save the info and break the loop.
				// There can never be a double check of 2 pawns.
				b.CheckInfo = pawnSq
				checkCounter++
				b.Squares[pawnSq.ToInfoIndex()].Set(INFO_MASK_CHECK)
				goto EXIT_PAWN_CHECK // Acts as double break.. nothing harmful really... :)
			}
		}
	}

EXIT_PAWN_CHECK:

	// 2.b No double check testing is needed. Pawn and knight cannot give double check.

	// 3. Detect checks (and pins) by Sliders.
	// 3.a) queens
	checkCounter += b.DetectSliderChecksAndPins(color, &pinMarker, checkCounter, &b.Queens[oppColor], QUEEN)
	if checkCounter > 1 {
		// Double check detected -> everything else is not of interest now.
		b.CheckInfo = CHECK_DOUBLE_CHECK
		goto PRETTY_PRINT_RETURN
	}
	// 3.b) bishops
	checkCounter += b.DetectSliderChecksAndPins(color, &pinMarker, checkCounter, &b.Bishops[oppColor], BISHOP)
	if checkCounter > 1 {
		// Double check detected -> everything else is not of interest now.
		b.CheckInfo = CHECK_DOUBLE_CHECK
		goto PRETTY_PRINT_RETURN
	}
	// 3.c) rooks
	checkCounter += b.DetectSliderChecksAndPins(color, &pinMarker, checkCounter, &b.Rooks[oppColor], ROOK)
	if checkCounter > 1 {
		// Double check detected -> everything else is not of interest now.
		b.CheckInfo = CHECK_DOUBLE_CHECK
		goto PRETTY_PRINT_RETURN
	}

PRETTY_PRINT_RETURN:
	//	fmt.Println(b.InfoBoardString())
	//	fmt.Println("CHECKINFO", b.CheckInfo)
	//	fmt.Println()
}

func (b *Board) DetectSliderChecksAndPins(color Color, pmarker *Info, ccount int, pieceList *PieceList, ptype Piece) int {
	kingSq := b.Kings[color]
	checkCounter := 0

	for i := uint8(0); i < pieceList.Size; i++ {
		sliderSq := pieceList.Pieces[i]
		diff := kingSq.Diff(sliderSq)
		diffTypes := SQUARE_DIFFS[diff]
		if diffTypes.Contains(ptype) {
			// The slider possibly checks the king or pins a piece.
			diffdir := DIFF_DIRS[diff]
			// Starting from the king we step through the path in question towards the enemy slider.
			info := INFO_NONE
			for stepSq := Square(int8(kingSq) + diffdir); ; stepSq = Square(int8(stepSq) + diffdir) {
				curPiece := b.Squares[stepSq]
				if curPiece.IsEmpty() {
					continue
				} else if curPiece.HasColor(color) {
					// A friendly piece was encountered
					if info == INFO_NONE {
						// This is the first piece on the path -> mark it as a possible pin.
						info.Set(INFO_MASK_MAYBE_PINNED)
					} else {
						// We already marked a piece on this path before -> there cannot be a pin anymore.
						// (There is an exception for EP capture but those are handled elsewhere.
						info = INFO_NONE
						break // Nothing more to do on this path.
					}
				} else {
					// An enemy piece was encountered
					if stepSq == sliderSq { //curPiece.Overlaps(diffTypes) {
						// We reached the checker/pinner -> decide.
						if info == INFO_NONE {
							// It's a check!
							checkCounter++
							b.CheckInfo = sliderSq
							info.Set(INFO_MASK_CHECK)
							break
						} else {
							// It's a pinner!
							info = *pmarker
							*pmarker++
							break
						}
					} else {
						// The enemy piece cannot pin or check. Break out.
						info = INFO_NONE
						break
					}
				}
			}

			if info.IsSet(INFO_MASK_CHECK) || (info.Pinval() != 0) {
				// We have detected a path that pins or checks. It will now
				// be marked in the info board, so move generation will skip
				// illegal moves.
				//				b.Squares[sliderSq.ToInfoIndex()] |= info
				for stepSq := sliderSq; stepSq != kingSq; stepSq = Square(int8(stepSq) - diffdir) {
					b.Squares[stepSq.ToInfoIndex()].Set(info)
				}
				// If it is a check we need to mark the square
				// which lies behind the king on the check-path.
				if info.IsSet(INFO_MASK_CHECK) {
					stepSq := Square(int8(kingSq) - diffdir)
					if stepSq.OnBoard() {
						b.Squares[stepSq.ToInfoIndex()].Set(INFO_MASK_FORBIDDEN_ESCAPE)
						//						tpiece := b.Squares[stepSq]
						//						if !tpiece.HasColor(color) {
						//							b.Squares[stepSq.ToInfoIndex()] = INFO_FORBIDDEN_ESCAPE
						//						}
					}
				}
			}

			if checkCounter+ccount > 1 {
				// Double check detected. Leave early.
				return checkCounter
			}
		}
	}

	return checkCounter
}

func (b *Board) GenerateAllLegalMoves(mlist *MoveList) {
	// Detect checks and pins.
	b.DetectChecksAndPins(b.Player)

	// Always generate king moves.
	b.GenerateKingMoves(mlist, b.Player)

	// If there is a double check skip generating other moves.
	if b.CheckInfo == CHECK_DOUBLE_CHECK {
		return
	} // Simple checks are handled by the following move generator functions.

	// Generate the rest of the legal moves.
	b.GenerateKnightMoves(mlist, b.Player)
	b.GenerateQueenMoves(mlist, b.Player)
	b.GenerateBishopMoves(mlist, b.Player)
	b.GenerateRookMoves(mlist, b.Player)
	b.GeneratePawnMoves(mlist, b.Player)
}

// GeneratePawnMoves generates all legal pawn moves for the given color
// and stores them in the given MoveList.
func (b *Board) GeneratePawnMoves(mlist *MoveList, color Color) {
	// NOTE: Pawn moves can be very complicated and have strange effects on the board (en passent, promotion..).
	// Because of this all pawn moves are tested for legality by 'fake-play'. This could be optimized by
	// testing the 'easy' ones differently (TODO).
	from, to := OTB, OTB
	tpiece := EMPTY
	oppColor := color.Flip()
	var move BitMove
	legal := false
	piece := PAWN | color

	// Possible en passent captures are detected backwards
	// so we do not need to add another conditional to the
	// capture loop below.
	if b.EpSquare != OTB {
		to = b.EpSquare
		// We use the 'wrong' color to find e.p. captures
		// by searching in the opposite direction.
		//		for _, dir := range PAWN_CAPTURE_DIRS[oppColor] {
		for d := 0; d < 2; d++ {
			dir := PAWN_CAPTURE_DIRS[oppColor][d]

			from = Square(int8(to) + dir)
			tpiece = b.Squares[from]
			if from.OnBoard() && tpiece == PAWN|color {
				move, legal = b.newPawnMoveIfLegal(color, from, to, piece, PAWN|oppColor, EMPTY, EP_TYPE_CAPTURE)
				if legal {
					mlist.Put(move)
				}
			}
		}
	}

	for i := uint8(0); i < b.Pawns[color].Size; i++ {
		// 0. Retrieve 'from' square from piece list.
		from = b.Pawns[color].Pieces[i]

		// a. Captures
		//		for _, capdir := range PAWN_CAPTURE_DIRS[color] {
		for d := 0; d < 2; d++ {
			capdir := PAWN_CAPTURE_DIRS[color][d]
			to = Square(int8(from) + capdir)
			// If the target square is on board and has the opponent's color
			// the capture is possible.
			if to.OnBoard() {
				tpiece = b.Squares[to]
				if tpiece.HasColor(oppColor) {
					// We also have to check if the capture is also a promotion.
					if to.IsPawnPromoting(color) {
						// If one type of promotion is legal, all are.
						legal = b.tryPawnMoveLegality(from, to, to, tpiece, color)
						if legal {
							for prom := QUEEN; prom >= KNIGHT; prom >>= 1 {
								move = NewBitMove(from, to, prom)
								mlist.Put(move)
							}
						}
					} else {
						move, legal = b.newPawnMoveIfLegal(color, from, to, piece, tpiece, EMPTY, EP_TYPE_NONE)
						if legal {
							mlist.Put(move)
						}
					}
				}
			}
		}

		// b. Push by one.
		// Target square does never need legality check here.
		to = Square(int8(from) + PAWN_PUSH_DIRS[color])
		if b.Squares[to].IsEmpty() {
			if to.IsPawnPromoting(color) {
				// If one type of promotion is legal, all are.
				legal = b.tryPawnMoveLegality(from, to, to, EMPTY, color)
				if legal {
					for prom := QUEEN; prom >= KNIGHT; prom >>= 1 {
						move = NewBitMove(from, to, prom)
						mlist.Put(move)
					}
				}

			} else {
				move, legal = b.newPawnMoveIfLegal(color, from, to, piece, EMPTY, EMPTY, EP_TYPE_NONE)
				if legal {
					mlist.Put(move)
				}
			}

			// c. Double push by advancing one more time, if the pawn was at base rank.
			if from.IsPawnBaseRank(color) {
				to = Square(int8(to) + PAWN_PUSH_DIRS[color])
				if b.Squares[to].IsEmpty() {
					move, legal = b.newPawnMoveIfLegal(color, from, to, piece, EMPTY, EMPTY, EP_TYPE_CREATE)
					if legal {
						mlist.Put(move)
					}
				}
			}
		}
	}

}

func (b *Board) newPawnMoveIfLegal(color Color, from, to Square, ptype, capPiece, promtype Piece, eptype uint8) (BitMove, bool) {
	capSq := to
	if eptype == EP_TYPE_CAPTURE {
		oppColor := color.Flip()
		// Find square where captured pawn is.
		capSq = Square(int8(to) + PAWN_PUSH_DIRS[oppColor])
	} else {
		frompinval := b.Squares[from.ToInfoIndex()].Pinval()
		isPinned := (frompinval != 0)
		if isPinned && b.Squares[to.ToInfoIndex()].Pinval() != frompinval {
			// Pinned and cannot move there.
			return BitMove(0), false
		} else if b.CheckInfo.OnBoard() && !b.Squares[to.ToInfoIndex()].IsSet(INFO_MASK_CHECK) {
			// Check and move doesn't change that.
			return BitMove(0), false
		} else {
			//			fmt.Println("PAWN MOVE", PrintBoardIndex[from], "->", PrintBoardIndex[to])
			return NewBitMove(from, to, promtype), true
		}
	}

	legal := b.tryPawnMoveLegality(from, to, capSq, capPiece, color)
	if legal {
		//		fmt.Println("PAWN MOVE", PrintBoardIndex[from], "->", PrintBoardIndex[to])
		return NewBitMove(from, to, promtype), true
	} else {
		return BitMove(0), false
	}

}

func (b *Board) tryPawnMoveLegality(from, to, capSq Square, capPiece Piece, color Color) bool {
	// Make the pawn move on the board but ignore possible promotions.
	b.Squares[capSq] = EMPTY
	b.Squares[to] = b.Squares[from] // Pawn is moved.

	b.Squares[from] = EMPTY

	// After the move test for check.
	legal := true
	if b.IsSquareAttacked(b.Kings[color], capSq, color) {
		legal = false
	}

	// Undo the fake move.
	b.Squares[from] = PAWN | color // Move back the pawn to where it came from.

	if capSq != to {
		b.Squares[capSq] = capPiece
		b.Squares[to] = EMPTY
	} else {
		b.Squares[to] = capPiece
	}

	return legal
}

// GenerateKnightMoves generates all legal knight moves for the given color
// and stores them in the given MoveList.
func (b *Board) GenerateKnightMoves(mlist *MoveList, color Color) {
	from, to := OTB, OTB
	tpiece := EMPTY
	var move BitMove

	isCheck := b.CheckInfo.OnBoard()

	// Iterate all knights of 'color'.
	for i := uint8(0); i < b.Knights[color].Size; i++ {
		from = b.Knights[color].Pieces[i]

		if b.Squares[from.ToInfoIndex()].Pinval() != 0 {
			// Pinned knights cannot move.
			continue
		}

		// Try all possible directions for a knight.
		//		for _, dir := range KNIGHT_DIRS {
		for d := 0; d < 8; d++ {
			dir := KNIGHT_DIRS[d]
			to = Square(int8(from) + dir)
			if to.OnBoard() {
				tpiece = b.Squares[to]
				if isCheck && !b.Squares[to.ToInfoIndex()].IsSet(INFO_MASK_CHECK) {
					// If there is a check but the move's target does not change that fact -> impossible move.
					continue
				} else if !tpiece.HasColor(color) {
					// Add a normal or a capture move.
					move = NewBitMove(from, to, NONE)
					mlist.Put(move)
				} // Else the square is occupied by own piece.
			} // Else target is outside of board.
		}
	}
}

// GenerateKingMoves generates all legal king moves for the given color
// and stores them in the given MoveList.
func (b *Board) GenerateKingMoves(mlist *MoveList, color Color) {
	from, to := b.Kings[color], OTB
	tpiece := EMPTY
	var move BitMove

	// Define all possible target squares for the king to move to and add all legal normal moves.
	targets := [8]bool{}

	//	for i, dir := range KING_DIRS {
	for i := 0; i < 8; i++ {
		dir := KING_DIRS[i]
		to = Square(int8(from) + dir)
		if to.OnBoard() {
			tpiece = b.Squares[to]
			if tpiece.HasColor(color) {
				continue
			}
			if b.IsSquareAttacked(to, OTB, color) {
				continue
			}
			targets[i] = true
			if !b.Squares[to.ToInfoIndex()].IsSet(INFO_MASK_FORBIDDEN_ESCAPE) {
				if !tpiece.HasColor(color) {
					// Add a normal or capture move.
					move = NewBitMove(from, to, NONE)
					mlist.Put(move)
				}
			}
		}
	}

	if b.CheckInfo != CHECK_NONE {
		// Cannot castle when in check.
		return
	}

	// a. Try castle king-side
	// Is it still allowed?
	if b.CastleShort[color] {
		sq1 := CASTLING_PATH_SHORT[color][0]
		sq2 := CASTLING_PATH_SHORT[color][1]

		// Test if the squares on short castling path are empty.
		sq1Piece := b.Squares[sq1]
		sq2Piece := b.Squares[sq2]
		if sq1Piece.IsEmpty() && sq2Piece.IsEmpty() {
			// Test if both squares are not attacked.
			if targets[0] && !b.IsSquareAttacked(sq2, OTB, color) {
				// Finally.. castling king-side is possible.
				move = NewBitMove(from, sq2, NONE)
				mlist.Put(move)
			}
		}
	}

	// b. Try castle queen-side
	// Is it still allowed?
	if b.CastleLong[color] {
		sq1 := CASTLING_PATH_LONG[color][0]
		sq2 := CASTLING_PATH_LONG[color][1]
		sq3 := CASTLING_PATH_LONG[color][2]

		// Test if the squares on short castling path are empty.
		sq1Piece := b.Squares[sq1]
		sq2Piece := b.Squares[sq2]
		sq3Piece := b.Squares[sq3]
		if sq1Piece.IsEmpty() && sq2Piece.IsEmpty() && sq3Piece.IsEmpty() {
			// Test if both squares are not attacked.
			if targets[1] && !b.IsSquareAttacked(sq2, OTB, color) {
				// Finally.. castling queen-side is possible.
				move = NewBitMove(from, sq2, NONE)
				mlist.Put(move)
			}
		}
	}
}

// GenerateBishopMoves generates all legal bishop moves for the given color
// and stores them in the given MoveList.
func (b *Board) GenerateBishopMoves(mlist *MoveList, color Color) {
	b.GenerateSlidingMoves(mlist, color, BISHOP, DIAGONAL_DIRS, &b.Bishops[color])
	//	from := OTB
	//	isCheck := b.CheckInfo.OnBoard()
	//	oppColor := color.Flip()

	//	// Iterate all bishop pieces.
	//	for i := uint8(0); i < b.Bishops[color].Size; i++ {
	//		from = b.Bishops[color].Pieces[i]
	//		isPinned := (b.Squares[from.ToInfoIndex()] >= INFO_PIN)

	//		// For every direction a bishop can go..
	//		for _, dir := range DIAGONAL_DIRS {
	//			b.generateInnerSlides(mlist, from, dir, isPinned, isCheck, color, oppColor)
	//		}
	//	}
}

// GenerateRookMoves generates all legal rook moves for the given color
// and stores them in the given MoveList.
func (b *Board) GenerateRookMoves(mlist *MoveList, color Color) {
	b.GenerateSlidingMoves(mlist, color, ROOK, ORTHOGONAL_DIRS, &b.Rooks[color])
	//	from := OTB
	//	isCheck := b.CheckInfo.OnBoard()
	//	oppColor := color.Flip()

	//	// Iterate all rook pieces.
	//	for i := uint8(0); i < b.Rooks[color].Size; i++ {
	//		from = b.Rooks[color].Pieces[i]
	//		isPinned := (b.Squares[from.ToInfoIndex()] >= INFO_PIN)

	//		// For every direction a rook can go..
	//		for _, dir := range ORTHOGONAL_DIRS {
	//			b.generateInnerSlides(mlist, from, dir, isPinned, isCheck, color, oppColor)
	//		}
	//	}
}

// GenerateQueenMoves generates all legal rook moves for the given color
// and stores them in the given MoveList.
func (b *Board) GenerateQueenMoves(mlist *MoveList, color Color) {
	b.GenerateSlidingMoves(mlist, color, QUEEN, ORTHOGONAL_DIRS, &b.Queens[color])
	b.GenerateSlidingMoves(mlist, color, QUEEN, DIAGONAL_DIRS, &b.Queens[color])
	//	from := OTB
	//	isCheck := b.CheckInfo.OnBoard()
	//	oppColor := color.Flip()

	//	// Iterate all queen pieces.
	//	for i := uint8(0); i < b.Queens[color].Size; i++ {
	//		from = b.Queens[color].Pieces[i]
	//		isPinned := (b.Squares[from.ToInfoIndex()] >= INFO_PIN)

	//		// For every direction a queen can go..
	//		for _, dir := range KING_DIRS {
	//			b.generateInnerSlides(mlist, from, dir, isPinned, isCheck, color, oppColor)
	//		}
	//	}
}

//func (b *Board) generateInnerSlides(mlist *MoveList, from Square, dir int8, isPinned, isCheck bool, color, oppColor Color) {
//	// -> repeat until a stop condition occurs.
//	for steps := int8(1); ; steps++ {
//		to := Square(int8(from) + dir*steps)

//		if !to.OnBoard() {
//			// Target is an illegal square -> next direction.
//			break
//		} else if isPinned && b.Squares[to.ToInfoIndex()] != b.Squares[from.ToInfoIndex()] {
//			// Piece is pinned but target is not on pin path -> next direction.
//			break
//		} else if isCheck && b.Squares[to.ToInfoIndex()] != INFO_CHECK {
//			tpiece := b.Squares[to]
//			if !tpiece.IsEmpty() {
//				// King is in check but the move does not capture the checker
//				// nor does it block the check -> skip direction.
//				break
//			}
//		} else {
//			tpiece := b.Squares[to]
//			if tpiece.HasColor(color) {
//				// Target is blocked by own piece -> next direction.
//				break
//			} else {
//				if tpiece.IsEmpty() {
//					// Add a normal move.
//					move := NewBitMove(from, to, NONE)
//					mlist.Put(move)
//					// And continue in current direction.
//				} else if tpiece.HasColor(oppColor) {
//					// Add a capture move.
//					move := NewBitMove(from, to, NONE)
//					mlist.Put(move)
//					// And go to next direction.
//					break
//				}
//			}
//		}

//	}
//}

// GenerateSlidingMoves generates all legal sliding moves for the given color
// and stores them in the given MoveList. This can be diagonal or orthogonal moves.
// This function is used to create all bishop, rook and queen moves.
func (b *Board) GenerateSlidingMoves(mlist *MoveList, color Color, ptype Piece, dirs [4]int8, plist *PieceList) {
	from, to := OTB, OTB
	tpiece := EMPTY
	var move BitMove
	oppColor := color.Flip()

	isCheck := b.CheckInfo.OnBoard()

	// Iterate all specific pieces.
	for i := uint8(0); i < plist.Size; i++ {
		from = plist.Pieces[i]
		isPinned := (b.Squares[from.ToInfoIndex()].Pinval() != 0)

		// For every direction a slider can go..
		//		for _, dir := range dirs {
		for i := 0; i < 4; i++ {
			dir := dirs[i]

			// -> repeat until a stop condition occurs.
			for steps := int8(1); ; steps++ {
				to = Square(int8(from) + dir*steps)

				if !to.OnBoard() {
					// Target is out of board -> next direction.
					break
				} else if isPinned && b.Squares[to.ToInfoIndex()] != b.Squares[from.ToInfoIndex()] {
					// Piece is pinned but target is not on pin path -> next direction.
					break
				} else if isCheck && !b.Squares[to.ToInfoIndex()].IsSet(INFO_MASK_CHECK) {
					tpiece = b.Squares[to]
					if !tpiece.IsEmpty() {
						// King is in check but the move does neither capture the checker
						// nor does it block the check -> skip direction.
						break
					}
				} else {
					tpiece = b.Squares[to]
					if tpiece.HasColor(color) {
						// Target is blocked by own piece -> next direction.
						break
					} else {
						if tpiece.IsEmpty() {
							// Add a normal move.
							move = NewBitMove(from, to, NONE)
							mlist.Put(move)
							// And continue in current direction.
						} else if tpiece.HasColor(oppColor) {
							// Add a capture move.
							move = NewBitMove(from, to, NONE)
							mlist.Put(move)
							// And go to next direction.
							break
						}
					}
				}
			}
		}

	}
}

func (b *Board) Perft(depth int) uint64 {
	mlist := MoveList{}
	cpy := *b
	nodes := uint64(0)

	if depth <= 0 {
		return 1
	}

	b.GenerateAllLegalMoves(&mlist)
	if depth == 1 {
		return uint64(mlist.Size)
	}

	for i := uint32(0); i < mlist.Size; i++ {
		move := &mlist.Moves[i]
		b.MakeLegalMove(*move)
		n := b.Perft(depth - 1)
		nodes += n

		*b = cpy //TODO --> better move undo
	}

	return nodes
}

func (b *Board) PerftDivide(depth int) map[string]uint64 {
	mlist := MoveList{}
	cpy := *b
	results := map[string]uint64{}

	b.GenerateAllLegalMoves(&mlist)

	for i := uint32(0); i < mlist.Size; i++ {
		move := &mlist.Moves[i]
		b.MakeLegalMove(*move)
		n := b.Perft(depth - 1)
		results[move.MiniNotation()] += n

		*b = cpy //TODO --> better move undo
	}

	return results
}

func (b *Board) InfoBoardString() string {
	str := "  +-----------------+\n"
	for r := 7; r >= 0; r-- {
		str += strconv.Itoa(r+1) + " | "
		for f := 0; f < 8; f++ {
			idx := 16*r + f + 8
			str += fmt.Sprintf("%d ", b.Squares[idx])
			//			str += strconv.Itoa(int(b.Squares[idx])) + " "
			if f == 7 {
				str += "|\n"
			}
		}
	}
	str += "  +-----------------+\n"
	str += "    a b c d e f g h\n"

	return str
}

func (b *Board) String() string {
	str := "  +-----------------+\n"
	for r := 7; r >= 0; r-- {
		str += strconv.Itoa(r+1) + " | "
		for f := 0; f < 8; f++ {
			idx := 16*r + f
			if Piece(idx) == b.EpSquare {
				str += ", "
			} else {
				str += PrintMap[b.Squares[idx]] + " "
			}

			if f == 7 {
				str += "|\n"
			}
		}
	}
	str += "  +-----------------+\n"
	str += "    a b c d e f g h\n"

	return str
}
