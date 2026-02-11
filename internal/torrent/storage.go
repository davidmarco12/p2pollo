package torrent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	g "github.com/anacrolix/generics"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
)

// fileStorage implementa storage.ClientImplCloser usando puro os.File
// sin mmap. Esto evita el error "MapViewOfFile: Not enough memory resources"
// en Windows con archivos grandes.
type fileStorage struct {
	baseDir    string
	completion storage.PieceCompletion
}

// NewFileStorage crea un storage basado en archivos sin mmap.
// Usa os.File.ReadAt/WriteAt en lugar de MapViewOfFile.
func NewFileStorage(baseDir string, completion storage.PieceCompletion) storage.ClientImplCloser {
	return &fileStorage{
		baseDir:    baseDir,
		completion: completion,
	}
}

func (fs *fileStorage) OpenTorrent(ctx context.Context, info *metainfo.Info, infoHash metainfo.Hash) (storage.TorrentImpl, error) {
	dir := filepath.Join(fs.baseDir, infoHash.HexString())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return storage.TorrentImpl{}, fmt.Errorf("crear directorio torrent: %w", err)
	}

	t := &fileTorrent{
		dir:        dir,
		info:       info,
		infoHash:   infoHash,
		completion: fs.completion,
	}

	return storage.TorrentImpl{
		Piece: func(p metainfo.Piece) storage.PieceImpl {
			return t.piece(p)
		},
		PieceWithHash: func(p metainfo.Piece, _ g.Option[[]byte]) storage.PieceImpl {
			return t.piece(p)
		},
		Close: t.close,
	}, nil
}

func (fs *fileStorage) Close() error {
	return fs.completion.Close()
}

// fileTorrent gestiona el almacenamiento de piezas para un torrent individual.
// Usa un solo archivo por torrent con offsets para cada pieza.
type fileTorrent struct {
	dir        string
	info       *metainfo.Info
	infoHash   metainfo.Hash
	completion storage.PieceCompletion

	mu   sync.Mutex
	file *os.File // archivo de datos, abierto lazy
}

func (t *fileTorrent) getFile() (*os.File, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.file != nil {
		return t.file, nil
	}

	path := filepath.Join(t.dir, "data")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("abrir archivo de datos: %w", err)
	}
	t.file = f
	return f, nil
}

func (t *fileTorrent) piece(p metainfo.Piece) storage.PieceImpl {
	return &filePiece{
		torrent: t,
		index:   p.Index(),
		offset:  p.Offset(),
		length:  p.Length(),
		key: metainfo.PieceKey{
			InfoHash: t.infoHash,
			Index:    p.Index(),
		},
	}
}

func (t *fileTorrent) close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.file != nil {
		err := t.file.Close()
		t.file = nil
		return err
	}
	return nil
}

// filePiece implementa storage.PieceImpl con puro ReadAt/WriteAt.
type filePiece struct {
	torrent *fileTorrent
	index   int
	offset  int64
	length  int64
	key     metainfo.PieceKey
}

func (p *filePiece) ReadAt(b []byte, off int64) (int, error) {
	f, err := p.torrent.getFile()
	if err != nil {
		return 0, err
	}
	return f.ReadAt(b, p.offset+off)
}

func (p *filePiece) WriteAt(b []byte, off int64) (int, error) {
	f, err := p.torrent.getFile()
	if err != nil {
		return 0, err
	}
	return f.WriteAt(b, p.offset+off)
}

func (p *filePiece) MarkComplete() error {
	return p.torrent.completion.Set(p.key, true)
}

func (p *filePiece) MarkNotComplete() error {
	return p.torrent.completion.Set(p.key, false)
}

func (p *filePiece) Completion() storage.Completion {
	c, err := p.torrent.completion.Get(p.key)
	if err != nil {
		return storage.Completion{Err: err}
	}
	return c
}
