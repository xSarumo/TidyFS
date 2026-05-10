<p align="center">
  <img src="https://github.com/xSarumo/xSarumo/blob/main/TidyFS/header.png" alt="TidyFS Header" width="100%">
</p>

---

**TidyFS** is a smart file organizer for Linux with a terminal user interface. It scans a folder with documents, classifies files by their content and filename, and then displays or organizes them into clear categories.

The project is written in **Go** and **Python**:

* Go is responsible for the TUI, file scanning, filesystem operations, and FUSE;
* Python is responsible for text extraction and ML-based document classification.

## Features

* Convenient TUI directly in the terminal.
* Automatic document classification by category.
* Supported modes:

  * `fuse` — virtual filesystem without moving files;
  * `move` — moves files into a sorted structure;
  * `copy` — copies files into a sorted structure.
* Supported document formats: `.pdf`, `.txt`, `.doc`, `.docx`, `.md`.
* Text preview extraction from PDF and DOCX files.
* Classification using an ML model based on TF-IDF features and Logistic Regression.
* Additional filename-based rules for common document types: resumes, certificates, lab reports, notes, licenses, tickets, statements, etc.
* Protection against dangerous directory cleanup: TidyFS refuses to clean `/`, the home directory, and conflicting source/target paths.

## Preview
<p align="center">
  <img src="https://github.com/xSarumo/xSarumo/blob/main/TidyFS/preview.png" alt="TidyFS Header" width="65%">
</p>

## How It Works

TidyFS follows a pipeline of several steps:

1. **Scanning**  
   The program walks through the source folder and collects a list of supported files.

2. **Text Extraction**  
   For text files, a small fragment of the content is extracted. For PDF and DOCX files, a Python extractor is used.

3. **Classification**  
   Each file is classified by its filename and content. The project uses a trained model stored in `classifier/models/classifier.joblib`.

4. **Organization**  
   Depending on the selected mode, TidyFS either mounts a virtual tree via FUSE or copies/moves files into the target directory.

Example output structure:

```text
~/TidyFS/
├── Career/
│   ├── Resume/
│   └── Internship/
├── Documents/
│   └── Personal/
├── Education/
│   ├── Assignments/
│   ├── Materials/
│   └── Project/
├── Finance/
├── IT/
│   └── Licenses/
└── Media/
    └── Transcript/
```

## Usage Examples

Run TidyFS:

```bash
tidyfs
```

Or run it from source:

```bash
make run
```

After launching the TUI, specify:

* **Source folder** — the folder to organize, for example `~/Downloads`;
* **Target folder** — the folder for the result, for example `~/TidyFS`;
* **Action mode** — the operation mode: `fuse`, `move`, or `copy`;
* **Clean target before work** — whether to clean the target directory before copying/moving files.

### `fuse` Mode

```text
Source: ~/Downloads
Target: ~/TidyFS
Mode:   fuse
```

In this mode, TidyFS creates a virtual representation of the files. The original files stay in place, while an organized category structure appears in the target folder.

This is the safest mode for the first run.

### `copy` Mode

```text
Source: ~/Downloads
Target: ~/TidyFS
Mode:   copy
```

TidyFS copies files from the source folder into the target folder, organizing them by category.

### `move` Mode

```text
Source: ~/Downloads
Target: ~/TidyFS
Mode:   move
```

TidyFS moves files from the source folder into the target folder. Use this mode only if you are sure the classification results are correct.

## Installation

### Requirements

* Linux
* Go
* Python 3
* `make`
* Python `venv`
* FUSE for the virtual filesystem mode

Python dependencies:

```text
scikit-learn
pypdf
python-docx
joblib
```

Go dependencies are installed through `go mod`.

## Installing Dependencies by Distribution

### Debian / Ubuntu

```bash
sudo apt update
sudo apt install -y git make golang python3 python3-venv python3-pip fuse3
```

### Fedora

```bash
sudo dnf install -y git make golang python3 python3-pip python3-virtualenv fuse3
```

### Arch Linux / Manjaro

```bash
sudo pacman -S --needed git make go python python-pip fuse3
```

### openSUSE

```bash
sudo zypper install git make go python3 python3-pip python3-virtualenv fuse3
```

## Building from Source

Clone the repository:

```bash
git clone https://github.com/<username>/TidyFS.git
cd TidyFS
```

Install dependencies and build the project:

```bash
make
```

This command runs:

```bash
make deps
make build
```

After building, the binary will be available here:

```text
bin/tidyfs
```

Run it:

```bash
./bin/tidyfs
```

## System Installation

The repository includes an `install.sh` script that builds the binary and copies it to `/usr/local/bin`:

```bash
chmod +x install.sh
./install.sh
```

After that, TidyFS can be launched from anywhere:

```bash
tidyfs
```

## Makefile Commands

```bash
make              # install dependencies and build the project
make deps         # install Go and Python dependencies
make go-deps      # run go mod tidy and go mod download
make py-venv      # create a Python virtual environment
make py-deps      # install Python dependencies
make build        # build the binary into bin/tidyfs
make run          # build and run the application
make dev          # install dependencies, build, and run
make clean        # remove bin/
make py-clean     # remove .venv/
make clean-all    # remove bin, .venv, pycache, and generated JSON files
```

## Project Structure

```text
TidyFS/
├── carrier/              # moving, copying, and FUSE file representation
├── classifier/           # Python classifier, model, and training data
│   ├── models/
│   │   └── classifier.joblib
│   ├── classifier.py
│   ├── train.ipynb
│   └── training_data.json
├── classifier_runner/    # running the Python classifier from Go
├── exporter/             # saving intermediate JSON files
├── extractor/            # extracting text from PDF/DOCX files
├── files/                # intermediate files: files.json and classified_files.json
├── project_path/         # project paths
├── scanner/              # scanning source directories
├── src/                  # Go application entry point
├── tui/                  # terminal user interface
├── Makefile
├── install.sh
├── go.mod
└── requirements.txt
```

## ML Classification

The classifier is located in `classifier/classifier.py`.

It uses:

* `TfidfVectorizer` for word-based features;
* `TfidfVectorizer` for character n-grams;
* `FeatureUnion` to combine features;
* `LogisticRegression` for final classification.

Before making an ML prediction, TidyFS also checks the filename using a set of regular expressions. This helps the system recognize obvious documents more accurately, such as resumes, certificates, lab reports, notes, licenses, and financial files.

## Supported Categories

Currently, TidyFS organizes files into the following main categories:

* `Career/Resume`
* `Career/Internship`
* `Documents/Personal`
* `Education/Assignments`
* `Education/Materials`
* `Education/Project`
* `Finance`
* `IT/Licenses`
* `Media/Transcript`

Some related categories are automatically merged. For example, `Education/Homework` and `Education/Labworks` are placed into `Education/Assignments`.

## Generated Files

During execution, TidyFS creates intermediate JSON files:

```text
files/files.json
files/classified_files.json
```

`files.json` contains discovered files and their text previews.  
`classified_files.json` contains classification results and is used by the FUSE/copy/move modes.

## Safety

TidyFS handles filesystem operations carefully:

* in `fuse` mode, original files are not moved;
* in `copy` mode, original files stay in place;
* in `move` mode, files are actually moved;
* when filenames conflict, TidyFS creates unique names such as `file (1).pdf`;
* target cleanup is protected against deleting the root directory, the home directory, and conflicting paths.

For the first run, it is recommended to use `fuse` or `copy` mode.

## Development

Install dependencies and run the development version:

```bash
make dev
```

Install Python dependencies separately:

```bash
make py-deps
```

Install Go dependencies separately:

```bash
make go-deps
```

Rebuild from scratch:

```bash
make rebuild
```

Remove all generated files:

```bash
make clean-all
```

## License

This project is distributed under the license specified in the [`LICENSE`](LICENSE) file.
