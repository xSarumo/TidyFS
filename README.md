<p align="center">
  <img src="https://github.com/xSarumo/xSarumo/blob/main/TidyFS/header.png" alt="TidyFS Header" width="100%">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/Python-3776AB?style=flat&logo=python&logoColor=white" alt="Python">
  <img src="https://img.shields.io/badge/Linux-FCC624?style=flat&logo=linux&logoColor=black" alt="Linux">
  <img src="https://img.shields.io/badge/FUSE-8A2BE2?style=flat" alt="FUSE">
  <img src="https://img.shields.io/badge/TF--IDF-9B5DE5?style=flat" alt="TF-IDF">
  <img src="https://img.shields.io/badge/LLM-Ollama-111827?style=flat" alt="Ollama LLM">
  <img src="https://img.shields.io/badge/Logistic_Regression-6C63FF?style=flat" alt="Logistic Regression">
</p>

---

# TidyFS

**TidyFS** is a smart file organizer for Linux with a terminal user interface. It scans a folder with documents, classifies files by their content and filename, and then displays or organizes them into clear categories.

The project is written in **Go** and **Python**:

* **Go** is responsible for the TUI, file scanning, filesystem operations, and FUSE;
* **Python** is responsible for text extraction and document classification;
* classification can run in two modes:

  * **TF-IDF fast** — local classic ML classifier;
  * **LLM slow** — local Ollama-powered classifier.

## Features

* Convenient TUI directly in the terminal.
* Automatic document classification by category.
* Classifier mode selection:

  * `TF_IDF` — faster, uses `classifier/models/classifier.joblib`;
  * `LLM` — slower, uses a local Ollama model.
* Supported action modes:

  * `fuse` — virtual filesystem without moving files;
  * `move` — moves files into a sorted structure;
  * `copy` — copies files into a sorted structure.
* Supported document formats: `.pdf`, `.txt`, `.doc`, `.docx`, `.md`.
* Text preview extraction from PDF and DOCX files.
* Filename-based rules for common document types: resumes, certificates, lab reports, notes, licenses, tickets, statements, etc.
* Protection against dangerous directory cleanup: TidyFS refuses to clean `/`, the home directory, and conflicting source/target paths.

## Preview

<p align="left">
  <img src="https://github.com/xSarumo/xSarumo/blob/main/TidyFS/preview.png" alt="TidyFS Preview" width="65%">
</p>

## How It Works

TidyFS follows a pipeline of several steps:

1. **Scanning**  
   The program walks through the source folder and collects a list of supported files.

2. **Text Extraction**  
   For text files, a small fragment of the content is extracted. For PDF and DOCX files, a Python extractor is used.

3. **Classification**  
   Each file is classified by its filename and content. TidyFS can use either the fast TF-IDF classifier or the slower local LLM classifier.

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
* **Action mode** — `fuse`, `move`, or `copy`;
* **Clean target before work** — whether to clean the target directory before copying/moving files;
* **Classifier** — `TF_IDF` or `LLM`.

### `fuse` Mode

```text
Source:     ~/Downloads
Target:     ~/TidyFS
Mode:       fuse
Classifier: TF_IDF fast
```

In this mode, TidyFS creates a virtual representation of the files. The original files stay in place, while an organized category structure appears in the target folder.

This is the safest mode for the first run.

### `copy` Mode

```text
Source:     ~/Downloads
Target:     ~/TidyFS
Mode:       copy
Classifier: LLM 
```

TidyFS copies files from the source folder into the target folder, organizing them by category.

### `move` Mode

```text
Source:     ~/Downloads
Target:     ~/TidyFS
Mode:       move
Classifier: TF_IDF fast
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
* Ollama, only if you want to use `LLM slow`

Python dependencies are installed from `requirements.txt`:

```text
scikit-learn
pypdf
python-docx
joblib
ollama
pydantic
```

> `pydantic` is required by the LLM classifier because it validates the JSON answer returned by the model.

Go dependencies are installed through `go mod`.

## Installing Dependencies by Distribution

### Debian / Ubuntu

```bash
sudo apt update
sudo apt install -y git make golang python3 python3-venv python3-pip fuse3 curl
```

### Fedora

```bash
sudo dnf install -y git make golang python3 python3-pip python3-virtualenv fuse3 curl
```

### Arch Linux / Manjaro

```bash
sudo pacman -S --needed git make go python python-pip fuse3 curl
```

### openSUSE

```bash
sudo zypper install git make go python3 python3-pip python3-virtualenv fuse3 curl
```

## Installing Ollama for LLM Classification

The `LLM slow` classifier uses a local Ollama server.

Install Ollama on Linux:

```bash
curl -fsSL https://ollama.com/install.sh | sh
```

Start the Ollama service:

```bash
sudo systemctl start ollama
sudo systemctl status ollama
```

Pull the default model used by TidyFS:

```bash
ollama pull qwen3:1.7b
```

You can also use another small local model:

```bash
ollama pull gemma3:1b
ollama pull qwen3:0.6b
```

Check installed models:

```bash
ollama list
```

Test the server:

```bash
ollama run qwen3:1.7b
```

or through the API:

```bash
curl http://localhost:11434/api/tags
```

## Building from Source

Clone the repository:

```bash
git clone https://github.com/xSarumo/TidyFS.git
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
├── classifier/           # Python classifiers, model, and training data
│   ├── models/
│   │   └── classifier.joblib
│   ├── classifier_tf_idf.py
│   ├── classifier_llm.py
│   ├── normalizer.py
│   ├── train.ipynb
│   └── training_data.json
├── classifier_runner/    # running the selected Python classifier from Go
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

## Classification Modes

TidyFS supports two classifier modes.

### TF-IDF Fast

`TF_IDF fast` runs:

```text
classifier/classifier_tf_idf.py
```

It uses:

* `TfidfVectorizer` for word-based features;
* `TfidfVectorizer` for character n-grams;
* `FeatureUnion` to combine features;
* `LogisticRegression` for final classification;
* filename regex rules before ML prediction.

This mode is fast and does not require Ollama.

### LLM

`LLM` runs:

```text
classifier/classifier_llm.py
```

It uses:

* local Ollama model;
* official `ollama` Python library;
* Pydantic validation;
* JSON Schema through `format=SCHEMA`;
* retry logic if the model returns invalid JSON;
* fallback category if all retries fail.

The answer is validated against the allowed category list. If the model returns an invalid category or invalid JSON, TidyFS retries.

## Changing the LLM Model

By default, the LLM classifier uses:

```python
model="qwen3:1.7b"
```

To change the model manually, edit this file:

```text
classifier/classifier_llm.py
```

Find:

```python
response = chat(
    model="qwen3:1.7b",
    messages=messages,
    options=options,
    format=SCHEMA
)
```

Replace it with another installed Ollama model, for example:

```python
response = chat(
    model="gemma3:1b",
    messages=messages,
    options=options,
    format=SCHEMA
)
```

Then pull the model:

```bash
ollama pull gemma3:1b
```

### Recommended: Use an Environment Variable

A more convenient approach is to make the model configurable through an environment variable.

In `classifier/classifier_llm.py`, add:

```python
import os

MODEL_NAME = os.getenv("TIDYFS_LLM_MODEL", "qwen3:1.7b")
```

Then change:

```python
model="qwen3:1.7b"
```

to:

```python
model=MODEL_NAME
```

Now you can run TidyFS with another model without editing code:

```bash
TIDYFS_LLM_MODEL=gemma3:1b make run
```

or, if installed globally:

```bash
TIDYFS_LLM_MODEL=gemma3:1b tidyfs
```

## Ollama Server Modes

Ollama has two common usage modes.

### Option 1: System Service

This is the default Linux installation style.

Start Ollama:

```bash
sudo systemctl start ollama
```

Enable it on boot:

```bash
sudo systemctl enable ollama
```

Stop it:

```bash
sudo systemctl stop ollama
```

View logs:

```bash
journalctl -e -u ollama
```

This option is simple and recommended if you often use LLM classification.

### Option 2: Start Ollama Only During LLM Classification

If you do not want Ollama to run all the time, TidyFS can start `ollama serve` only when `LLM slow` is selected, then stop it after classification.

Recommended behavior:

1. If Ollama is already running, TidyFS uses it and does not stop it.
2. If Ollama is not running, TidyFS starts `ollama serve`.
3. After LLM classification finishes, TidyFS stops only the server process it started itself.

Add this logic in Go inside `classifier_runner/classifier_runner.go`, because this package already decides which Python classifier to run.

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

## Troubleshooting

### `ollama not found in PATH`

Install Ollama:

```bash
curl -fsSL https://ollama.com/install.sh | sh
```

Then check:

```bash
ollama --version
```

### `connection refused` or Ollama is not running

Start Ollama manually:

```bash
sudo systemctl start ollama
```

or use the app-managed `ollama serve` approach described above.

### Model not found

Pull the model:

```bash
ollama pull qwen3:1.7b
```

or change `TIDYFS_LLM_MODEL` to a model that exists locally:

```bash
ollama list
```

### Python dependency error

Reinstall Python dependencies:

```bash
make py-clean
make py-deps
```

Make sure `requirements.txt` contains:

```text
pydantic
ollama
```

## License

This project is distributed under the MIT License.
