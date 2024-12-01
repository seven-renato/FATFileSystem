package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"unsafe"
)

func main() {
	// fmt.Printf("O tamanho de FATEntry e %d bytes \n", unsafe.Sizeof(FATEntry{})) -> 12 bytes, compilador adiciona 3 bytes apos o campo USED para alinha ao tamanho com os outros campos -> Facilita a busca e acesso em memoria
	fsSize := getFileSystemSize()
	if fsSize == 0 {
		return
	}
	var blockSize uint32 = 4096
	fs, err := createFileSystem(blockSize, fsSize)
	if err != nil {
		return
	}
	operateFileSystem(fs)
}

// Create File System

func getFileSystemSize() uint32 {
	var size uint32
	running := true
	for running {
		fmt.Println("Escolha sua opção:")
		fmt.Println("1. 10MB")
		fmt.Println("2. 100MB")
		fmt.Println("3. 800MB")
		fmt.Println("4. Sair.")
		consoleScanner := bufio.NewScanner(os.Stdin)
		fmt.Printf("Resposta: ")
		consoleScanner.Scan()
		inputStr := consoleScanner.Text()
		option, e := strconv.Atoi(inputStr)
		if e != nil {
			fmt.Printf("Entrada inválida: '%s'. Por favor, insira um número entre 1 e 4.\n", inputStr)
			continue
		}
		switch option {
		case 1:
			size = 10 * 1024 * 1024
		case 2:
			size = 100 * 1024 * 1024
		case 3:
			size = 800 * 1024 * 1024
		case 4:
			running = false
			continue
		default:
			fmt.Println("Opção inválida. Escolha um número entre 1 e 4.")
		}
		return size
	}
	return 0
}

type Header struct {
	TotalSize            uint32
	BlockSize            uint32
	FreeSpace            uint32
	FATEntrypointAddress uint32
	RootDirStart         uint32
	DataStart            uint32
}

type FATEntry struct {
	BlockID     uint32 // 4 bytes de 0 a 2**32 - 1
	NextBlockID uint32 // 4 bytes
	Used        bool   // 1 byte
}

type FileEntry struct {
	Name         [32]byte
	Size         uint32
	FirstBlockID uint32
	Protected    bool
}
type FURGFileSystem struct {
	Header      Header
	FAT         []FATEntry
	RootDir     []FileEntry
	FilePointer *os.File
}

func calculateFATSize(FileSystemSize uint32, BlockSize uint32, FATEntrySize uint32) uint32 {
	totalBlocks := FileSystemSize / BlockSize
	fatSize := totalBlocks * FATEntrySize
	return fatSize
}

func calculateRootDirSize(entriesNumber uint32) uint32 {
	rootDirSize := uint32(entriesNumber) * uint32(unsafe.Sizeof(FileEntry{}))
	return rootDirSize
}

func calculateHeaderSize() uint32 {
	HeaderSize := uint32(unsafe.Sizeof(Header{}))
	return HeaderSize
}

func createFileSystem(BlockSize uint32, TotalSize uint32) (*FURGFileSystem, error) {
	f, err := os.OpenFile("furg.fs2", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("Erro ao abrir/criar o arquivo", err)
		return nil, nil
	} else {
		fmt.Println("Arquivo do FileSystem criado com sucesso com permissao de escrita e leitura.")
	}

	var entriesNumber uint32 = 100

	rootDirSize := calculateRootDirSize(entriesNumber)
	headerSize := calculateHeaderSize()
	fatEntrySize := uint32(unsafe.Sizeof(FATEntry{}))
	FATSize := calculateFATSize(TotalSize-headerSize-rootDirSize, BlockSize, fatEntrySize)

	header := Header{
		TotalSize:            TotalSize,
		BlockSize:            BlockSize,
		FreeSpace:            TotalSize - headerSize - FATSize - rootDirSize,
		FATEntrypointAddress: headerSize,
		RootDirStart:         headerSize + FATSize,
		DataStart:            headerSize + FATSize + rootDirSize,
	}

	err = binary.Write(f, binary.LittleEndian, header)
	if err != nil {
		fmt.Println("Escrita do arquivo em binario falhou.", err)
	}
	fileSystem := FURGFileSystem{
		Header:      header,
		FAT:         make([]FATEntry, FATSize/fatEntrySize),
		RootDir:     make([]FileEntry, entriesNumber),
		FilePointer: f,
	}

	return &fileSystem, nil // Retornar pontiero pois ao inves de duplicar a memoria, apenas retorna o ponteiro de referencia a ele.
}

// Operate in File System

func operateFileSystem(FileSystem *FURGFileSystem) {
	var option int
	for {
		fmt.Println("\n--- Menu do Sistema de Arquivos FURGfs2 ---")
		fmt.Println("1. Copiar arquivo para o sistema de arquivos")
		fmt.Println("2. Remover arquivo do sistema de arquivos")
		fmt.Println("3. Renomear arquivo armazenado no FURGfs2")
		fmt.Println("4. Listar todos os arquivos armazenados no FURGfs2")
		fmt.Println("5. Listar o espaço livre em relação ao total do FURGfs2")
		fmt.Println("6. Proteger/desproteger arquivo contra escrita/remoção")
		fmt.Println("0. Sair")
		fmt.Print("Escolha uma opção: ")
		fmt.Scanln(&option)

		switch option {
		case 1:
			fmt.Println("Opção 1: Copiar arquivo para o sistema de arquivos.")
			fmt.Print("Digite o caminho completo do arquivo para copiar: ")
			var path string
			fmt.Scanln(&path)

			fmt.Print("Digite o bit de proteção (1 para protegido, 0 para não protegido): ")
			var protectionBit int
			fmt.Scanln(&protectionBit)
			if protectionBit != 0 && protectionBit != 1 {
				fmt.Println("Bit de proteção inválido! Deve ser 1 ou 0.")
				continue
			}
			isProtected := protectionBit == 1

			fmt.Printf("Arquivo '%s' será copiado com proteção: %d\n", path, protectionBit)
			copyFileToFileSystem(FileSystem, path, isProtected)
		case 2:
			fmt.Println("Opção 2: Remover arquivo do sistema de arquivos.")
			fmt.Print("Digite o caminho completo do arquivo para remover: ")
			var path string
			fmt.Scanln(&path)
			fmt.Printf("Arquivo '%s' será removido.\n", path)

		case 3:
			fmt.Println("Opção 3: Renomear arquivo armazenado no FURGfs2.")
			fmt.Print("Digite o nome do arquivo a ser renomeado: ")
			var oldName string
			fmt.Scanln(&oldName)
			fmt.Print("Digite o novo nome do arquivo: ")
			var newName string
			fmt.Scanln(&newName)
			fmt.Printf("Arquivo '%s' será renomeado para '%s'.\n", oldName, newName)

		case 4:
			fmt.Println("Opção 4: Listar todos os arquivos armazenados no FURGfs2.")
			fmt.Println("Listagem de arquivos...")

		case 5:
			fmt.Println("Opção 5: Listar o espaço livre em relação ao total do FURGfs2.")
			fmt.Println("Espaço livre e total: ...")

		case 6:
			fmt.Println("Opção 6: Proteger/desproteger arquivo contra escrita/remoção.")
			fmt.Print("Digite o nome do arquivo a ser protegido/desprotegido: ")
			var fileName string
			fmt.Scanln(&fileName)
			fmt.Print("Deseja proteger (1) ou desproteger (0) o arquivo? ")
			var action string
			fmt.Scanln(&action)
			if action != "1" && action != "0" {
				fmt.Println("Ação inválida! Deve ser '1' para proteger ou '0' para desproteger.")
				continue
			}
			fmt.Printf("Arquivo '%s' será %s.\n", fileName, map[string]string{"1": "protegido", "0": "desprotegido"}[action])

		case 0:
			fmt.Println("Saindo do sistema. Até logo!")
			return

		default:
			fmt.Println("Opção inválida. Tente novamente.")
		}
	}
}

func checkFileNameAlreadExists(filename [32]byte, FileSystem *FURGFileSystem) int {
	for i, v := range FileSystem.RootDir {
		if v.Name == filename {
			return i
		}
	}
	return 0
}

func processFileForFileSystem(FileSystem *FURGFileSystem, path string) (*os.File, [32]byte, string, uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, [32]byte{}, "", 0, fmt.Errorf("erro ao abrir o arquivo: %w", err)
	}

	fileInfo, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, [32]byte{}, "", 0, fmt.Errorf("erro ao obter informações do arquivo: %w", err)
	}

	fileSize := fileInfo.Size()
	if fileSize > int64(FileSystem.Header.FreeSpace) {
		f.Close()
		return nil, [32]byte{}, "", 0, fmt.Errorf("erro: o arquivo é muito grande para o espaço disponível")
	}

	var fileSizeUint32 uint32 = uint32(fileSize)

	fileName := filepath.Base(path)

	if len(fileName) > 32 {
		f.Close()
		return nil, [32]byte{}, "", 0, fmt.Errorf("erro: o nome do arquivo excede o limite de 32 bytes")
	}

	var fileNameArray [32]byte
	copy(fileNameArray[:], fileName)

	return f, fileNameArray, fileName, fileSizeUint32, nil
}

func copyFileToFileSystem(FileSystem *FURGFileSystem, Path string, Protected bool) bool {
	f, fileNameArray, fileName, fileSizeUint32, err := processFileForFileSystem(FileSystem, Path)

	if err != nil {
		fmt.Println(err)
		return false
	}

	if checkFileNameAlreadExists(fileNameArray, FileSystem) != 0 {
		fmt.Printf("O arquivo com nome '%s' já foi armazenado no sistema de arquivos.", fileName)
		return false
	}

	copy(fileNameArray[:], fileName)

	buf := make([]byte, FileSystem.Header.BlockSize)

	var firstBlock, previousBlock uint32
	firstBlockSet := false
	for {
		bytesRead, err := f.Read(buf)
		if err != nil && err != io.EOF {
			fmt.Println("Erro ao ler o arquivo:", err)
			return false
		}
		if bytesRead == 0 {
			break
		}

		var currentBlockID uint32
		found := false
		for i, v := range FileSystem.FAT {
			if v.Used == false {
				currentBlockID = uint32(i)
				tmp := FATEntry{
					BlockID:     currentBlockID,
					NextBlockID: 0,
					Used:        true,
				}
				found = true
				FileSystem.FAT[i] = tmp
				break
			}
		}
		if !found {
			fmt.Println("Erro: espaço insuficiente na FAT.")
			return false
		}

		if !firstBlockSet {
			firstBlock = currentBlockID
			firstBlockSet = true
		} else {
			FileSystem.FAT[previousBlock].NextBlockID = currentBlockID
		}
		previousBlock = currentBlockID

		_, err = FileSystem.FilePointer.Seek(int64(FileSystem.Header.DataStart+(currentBlockID*FileSystem.Header.BlockSize)), 0)
		if err != nil {
			fmt.Println("Erro ao mover ponteiro do arquivo:", err)
			return false
		}
		_, err = FileSystem.FilePointer.Write(buf[:bytesRead])
		if err != nil {
			fmt.Println("Erro ao escrever dados no arquivo:", err)
			return false
		}
	}

	for i, entry := range FileSystem.RootDir {
		if entry.Name[0] == 0 {
			FileSystem.RootDir[i] = FileEntry{
				Name:         fileNameArray,
				Size:         fileSizeUint32,
				FirstBlockID: firstBlock,
				Protected:    Protected,
			}
			break
		}
	}
	FileSystem.Header.FreeSpace -= fileSizeUint32
	fmt.Printf("Arquivo '%s' copiado com sucesso para o sistema de arquivos.\n", fileName)
	return true
}

func removeFileFromFileSystem(FileSystem *FURGFileSystem, Path string) bool {
	_, fileNameArray, fileName, _, err := processFileForFileSystem(FileSystem, Path)

	if err != nil {
		fmt.Println(err)
		return false
	}

	rootDirIndex := checkFileNameAlreadExists(fileNameArray, FileSystem)
	if rootDirIndex == 0 {
		fmt.Printf("O arquivo com nome '%s' não foi armazenado no sistema de arquivos.\n", fileName)
		return false
	}

	file := FileSystem.RootDir[rootDirIndex]

	nextBlockId := file.FirstBlockID
	for nextBlockId != 0 {
		currentFileEntry := FileSystem.FAT[nextBlockId]
		currentFileEntry.Used = false
		nextBlockId = currentFileEntry.NextBlockID
	}

	FileSystem.RootDir[rootDirIndex] = FileEntry{}

	fmt.Printf("O arquivo com nome '%s' foi removido no sistema de arquivos.\n", fileName)
	return true

}
