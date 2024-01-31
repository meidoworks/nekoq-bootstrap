package wal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/afero"

	"github.com/meidoworks/nekoq-bootstrap/internal/iface"
)

type EntryType int

const (
	EntryTypeWhole     EntryType = 0b00001001
	EntryTypeStart     EntryType = 0b00001010
	EntryTypeMiddle    EntryType = 0b00001011
	EntryTypeEnd       EntryType = 0b00001100
	EntryTypeBrokenEnd EntryType = 0b00001101

	EntryTypeMask = 0b00001111
)

const (
	EntryHeader      int = 0b10100000
	RecordHeaderSize     = 8
)

var (
	StartWalSequence = iface.SequenceId{Term: 1, Collection: 1, Seq: 1}
)

var _ iface.Wal = new(DiskWal)

type DiskWal struct {
	curTerm       int64 // term
	curCollection int64 // file
	curSeq        int64 // sequence inside a collection
	pageOccupied  int32

	maxFileSize int64
	maxPageSize int32
	folder      string

	curFile *os.File

	sync.Mutex
	dirty        int32
	closeChannel chan struct{}
}

func NewDiskWal(folder string) *DiskWal {
	// page structure: 4K { record1, record2, ..., padding(0) }
	// record structure: { Entry header | EntryType = 1B + record size = 2B + page crc32 = 4B + reserved =1B }
	return &DiskWal{
		maxFileSize:  1 * 1024 * 1024 * 1024,
		maxPageSize:  4 * 1024,
		folder:       folder,
		curTerm:      1, // default term, will be soon overwritten by Initialize() or external term
		closeChannel: make(chan struct{}),
	}
}

func (d *DiskWal) WriteEntry(bytes []byte) (iface.SequenceId, error) {
	return d.WriteEntryWithFullPage(bytes)
}

func (d *DiskWal) WriteEntryWithFullPage(bytes []byte) (iface.SequenceId, error) {
	d.Lock()
	defer d.Unlock()

	var recordMax = int(d.maxPageSize - RecordHeaderSize)

	if recordMax >= len(bytes) {
		// within one page
		if err := d.guaranteePage(1); err != nil {
			return iface.SequenceId{}, err
		}
		if err := d.writeDataPage(bytes[:], EntryTypeWhole); err != nil {
			return iface.SequenceId{}, err
		}
	} else {
		// multiple pages
		var offset = 0
		var n = 0
		var t = EntryTypeStart
		for {
			if offset == 0 {
				t = EntryTypeStart
			} else {
				t = EntryTypeMiddle
			}
			if offset+recordMax >= len(bytes) {
				n = len(bytes) - offset
				t = EntryTypeEnd
			} else {
				n = recordMax
			}
			if err := d.guaranteePage(1); err != nil {
				return iface.SequenceId{}, err
			}
			if err := d.writeDataPage(bytes[offset:offset+n], t); err != nil {
				return iface.SequenceId{}, err
			}
			offset += n
			if offset >= len(bytes) {
				break
			}
		}
	}

	// mark dirty
	atomic.StoreInt32(&d.dirty, 1)
	return iface.SequenceId{
		Term:       d.curTerm,
		Collection: d.curCollection,
		Seq:        d.curSeq,
	}, nil
}

func (d *DiskWal) filepath(term, collection, seq int64) string {
	return filepath.Join(d.folder, fmt.Sprintf("%016x%016x", term, collection))
}

func (d *DiskWal) filename(term, collection, seq int64) string {
	return fmt.Sprintf("%016x%016x", term, collection)
}

func (d *DiskWal) guaranteePage(pageCnt int) error {
	// no file open
	if d.curFile == nil {
		d.curCollection++
		d.curSeq = 0
		f, err := os.OpenFile(d.filepath(d.curTerm, d.curCollection, d.curSeq), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0664)
		if err != nil {
			return err
		}
		d.curFile = f
		return nil
	}
	// reach max file size
	if int64((int(d.pageOccupied)+pageCnt)*int(d.maxPageSize)) > d.maxFileSize {
		if err := d.curFile.Close(); err != nil {
			return err
		}
		d.curCollection++
		d.curSeq = 0
		d.pageOccupied = 0
		f, err := os.OpenFile(d.filepath(d.curTerm, d.curCollection, d.curSeq), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0664)
		if err != nil {
			return err
		}
		d.curFile = f
	}
	return nil
}

func (d *DiskWal) writeDataPage(dat []byte, entryType EntryType) error {
	buf := d.prepareDataBufByPage(dat, entryType)
	if _, err := d.curFile.Write(buf); err != nil {
		return err
	}
	d.curSeq++
	d.pageOccupied++
	return nil
}

func (d *DiskWal) prepareDataBufByPage(dat []byte, entryType EntryType) []byte {
	buf := make([]byte, d.maxPageSize)
	buf[0] = byte(EntryHeader) | byte(entryType)
	binary.BigEndian.PutUint16(buf[1:3], uint16(len(dat)))
	copy(buf[8:], dat)
	binary.BigEndian.PutUint32(buf[3:7], crc32.Checksum(buf, crc32.IEEETable))
	return buf
}

func (d *DiskWal) CurrentSequence() iface.SequenceId {
	d.Lock()
	defer d.Unlock()

	return d.curPosition()
}

func (d *DiskWal) listWalFiles() ([]string, error) {
	var fileNames []string
	afs := afero.NewBasePathFs(afero.NewOsFs(), d.folder)
	if err := afero.Walk(afs, "", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if name, _, err := d.walfilename(path); err != nil {
			log.Println("list wal file - found unknown file:", path, "with error:", err)
			return nil
		} else {
			fileNames = append(fileNames, name)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	slices.Sort(fileNames)
	return fileNames, nil
}

func (d *DiskWal) Initialize() (iface.SequenceId, error) {
	d.Lock()
	defer d.Unlock()

	if err := afero.NewOsFs().MkdirAll(d.folder, 0755); err != nil {
		return iface.SequenceId{}, err
	}
	fileMap := map[string]iface.SequenceId{}
	afs := afero.NewBasePathFs(afero.NewOsFs(), d.folder)
	if err := afero.Walk(afs, "", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if name, seq, err := d.walfilename(path); err != nil {
			log.Println("Initialize DiskWal - found unknown file:", path, "with error:", err)
			return nil
		} else {
			fileMap[name] = seq
		}
		return nil
	}); err != nil {
		return iface.SequenceId{}, err
	}
	var fileNames []string
	for key, _ := range fileMap {
		fileNames = append(fileNames, key)
	}
	slices.Sort(fileNames)
	// empty
	if len(fileNames) == 0 {
		d.curTerm = 1
		d.curCollection = 0
		d.curSeq = 0
		return d.curPosition(), nil
	}

	// 1. read SequenceId and last file
	//    + fix incomplete wal(no EntryTypeEnd) with EntryTypeBrokenEnd in order to indicate that the whole record is invalid
	lastFileName := fileNames[len(fileNames)-1]
	finfo, err := d.readWalFileInfo(afs, lastFileName)
	if err != nil {
		return iface.SequenceId{}, err
	}
	d.curSeq = finfo.totalValidSeq
	d.curTerm = fileMap[lastFileName].Term
	d.curCollection = fileMap[lastFileName].Collection
	// 2.open latest wal file and seek at insert point
	if err := d.seekWalFileWritePoint(lastFileName); err != nil {
		return iface.SequenceId{}, err
	}
	// 3. start background file sync work
	go func() {
		for {
			select {
			case <-d.closeChannel:
			default:
				time.Sleep(200 * time.Millisecond)
				if atomic.LoadInt32(&d.dirty) == 1 {
					for !atomic.CompareAndSwapInt32(&d.dirty, 1, 0) {
					}
					if err := d.curFile.Sync(); err != nil {
						log.Println("wal background file sync failed:", err)
					}
				}
			}
		}
	}()
	return iface.SequenceId{
		Term: d.curTerm, Collection: d.curCollection, Seq: d.curSeq,
	}, nil
}

func (d *DiskWal) walfilename(name string) (string, iface.SequenceId, error) {
	if len(name) != 32 {
		return "", iface.SequenceId{}, errors.New("file name length mismatch")
	}
	var v1 int64
	var v2 int64
	_, err := fmt.Sscanf(name, "%016x%016x", &v1, &v2)
	if err != nil {
		return "", iface.SequenceId{}, err
	}
	if v1 <= 0 || v2 <= 0 {
		return "", iface.SequenceId{}, errors.New("file sequence id has one or more less than 1")
	}
	return name, iface.SequenceId{
		Term:       v1,
		Collection: v2,
		Seq:        0, // not read file yet, 0 as default
	}, nil
}

func (d *DiskWal) readWalFileInfo(afs afero.Fs, file string) (result struct {
	totalPages    int32
	totalValidSeq int64
	clean         bool
}, rerr error) {
	pageBuf := make([]byte, d.maxPageSize)
	if f, e := afs.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0664); e != nil {
		rerr = e
		return
	} else {
		defer func(f afero.File) {
			_ = f.Close()
		}(f)
		var lastRecordsInfo []struct {
			Data []byte
			Type EntryType
		}
		for {
			_, err := io.ReadFull(f, pageBuf)
			if err == io.EOF {
				break
			} else if err != nil {
				rerr = err
				return
			}
			recordsInfo, err := d.pageRecordsInfo(pageBuf)
			if err != nil {
				rerr = err
				return
			}
			lastRecordsInfo = recordsInfo
			result.totalPages++
			result.totalValidSeq += int64(len(recordsInfo))
		}
		// last page
		if len(pageBuf) <= 0 {
			// empty file
			result.totalPages = 0
			result.totalValidSeq = 0
			result.clean = true
			return
		} else {
			if lastRecordsInfo[len(lastRecordsInfo)-1].Type == EntryTypeMiddle {
				// incomplete record, write broken end
				dat := d.prepareDataBufByPage(nil, EntryTypeBrokenEnd)
				if _, err := f.Write(dat); err != nil {
					rerr = err
					return
				}
				result.totalPages++
				result.totalValidSeq++
			}
			return
		}
	}
}

func (d *DiskWal) pageRecordsInfo(buf []byte) ([]struct {
	Data []byte
	Type EntryType
}, error) {
	var crc32dat [4]byte
	crc32dat[0] = buf[3]
	crc32dat[1] = buf[4]
	crc32dat[2] = buf[5]
	crc32dat[3] = buf[6]
	clear(buf[3:7])
	crc32val := binary.BigEndian.Uint32(crc32dat[:])
	crc32nval := crc32.Checksum(buf, crc32.IEEETable)
	if crc32nval != crc32val {
		return nil, errors.New("page crc32 mismatch")
	}
	length := binary.BigEndian.Uint16(buf[1:3])
	if int(length+8) > len(buf) {
		return nil, errors.New("length field exceeded")
	}
	var r []struct {
		Data []byte
		Type EntryType
	}
	r = append(r, struct {
		Data []byte
		Type EntryType
	}{
		Type: EntryType(buf[0] & EntryTypeMask),
		Data: buf[8 : 8+length],
	})
	return r, nil
}

func (d *DiskWal) curPosition() iface.SequenceId {
	return iface.SequenceId{
		Term:       d.curTerm,
		Collection: d.curCollection,
		Seq:        d.curSeq,
	}
}

func (d *DiskWal) seekWalFileWritePoint(filename string) error {
	fp, err := filepath.Abs(filepath.Join(d.folder, filename))
	if err != nil {
		return err
	}
	f, err := os.OpenFile(fp, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		return err
	}
	{
		buf := make([]byte, d.maxPageSize)
		for {
			_, err := io.ReadFull(f, buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			d.pageOccupied++
		}
	}
	d.curFile = f
	return nil
}

func (d *DiskWal) ReplayIncomplete(from iface.SequenceId, f func(entry []byte) error) (iface.SequenceId, error) {
	d.Lock()
	defer d.Unlock()

	if d.curTerm == from.Term && d.curCollection == from.Collection && d.curSeq == from.Seq {
		return from, nil
	}
	// no wal
	if d.curTerm == 1 && d.curCollection == 0 && d.curSeq == 0 {
		return StartWalSequence, nil
	}
	if from.Compare(iface.SequenceId{
		Term:       d.curTerm,
		Collection: d.curCollection,
		Seq:        d.curSeq,
	}) > 0 {
		return iface.SequenceId{}, errors.New("replay incomplete failed - request newer wal than history:" + fmt.Sprint(from))
	}
	files, err := d.listWalFiles()
	if err != nil {
		return iface.SequenceId{}, err
	}
	var targetFile = d.filename(from.Term, from.Collection, from.Seq)
	var idx = -1
	for _, v := range files {
		if targetFile < v {
			break
		} else if targetFile >= v {
			idx++
		}
	}
	if idx == -1 {
		return iface.SequenceId{}, errors.New("replay incomplete failed - insufficient wal history, require:" + fmt.Sprint(from))
	}
	for i := idx; i < len(files); i++ {
		if err := d.replayFile(files[i], f); err != nil {
			return iface.SequenceId{}, err
		}
	}
	return iface.SequenceId{
		Term:       d.curTerm,
		Collection: d.curCollection,
		Seq:        d.curSeq,
	}, nil
}

func (d *DiskWal) replayFile(filename string, fn func(entry []byte) error) error {
	pageBuf := make([]byte, d.maxPageSize)
	if f, err := os.OpenFile(filepath.Join(d.folder, filename), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0664); err != nil {
		return err
	} else {
		defer func(f afero.File) {
			_ = f.Close()
		}(f)
		for {
			_, err := io.ReadFull(f, pageBuf)
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}
			recordsInfo, err := d.pageRecordsInfo(pageBuf)
			if err != nil {
				return err
			}
			for _, v := range recordsInfo {
				if err := fn(v.Data); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
