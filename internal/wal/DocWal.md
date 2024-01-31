# Wal

## 1. Design

### 1.1 Prerequisite

* Assume files are complete, no loss
* (Current implementation)Always full page write
* (Current implementation)Manual clean unused wal files

### 1.2 Scenarios

#### 1.2.1 Fresh startup

#### 1.2.2 Startup from existing files, no data integrity issue

#### 1.2.3 Startup from existing files, with data integrity issue

#### 1.2.4 Write within same file and one page

#### 1.2.5 Write within same file and beyond one page

#### 1.2.6 Write across multiple files(file rotate)

## 2. Planning

* Non-full-page write
* Wal file auto clean
* Discard crc32 failed pages
