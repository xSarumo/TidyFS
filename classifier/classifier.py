import json
import re
import sys
import joblib
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(PROJECT_ROOT))

from extractor.extractor import extract_preview

FILES_PATH = PROJECT_ROOT / "files" / "files.json"
OUTPUT_PATH = PROJECT_ROOT / "files" / "classified_files.json"
MODEL_PATH = PROJECT_ROOT / "classifier" / "models" / "classifier.joblib"

CONFIDENCE_THRESHOLD = 0.0

CATEGORY_MERGE_MAP = {
    "Education/Homework": "Education/Assignments",
    "Education/Labworks": "Education/Assignments",

    "Education/Certificates": "Documents/Personal",
    "Personal/Documents": "Documents/Personal",

    "Media/Books": "Education/Materials",
    "Education/Materials": "Education/Materials",

    "Career/Internship": "Career/Internship",
    "Career/Resume": "Career/Resume",
    "Data/Datasets": "Education/Materials",
    "Education/Project": "Education/Project",
    "Finance": "Finance",
    "IT/Licenses": "IT/Licenses",
    "Media/Transcript": "Media/Transcript",
}


def merge_category(category: str) -> str:
    return CATEGORY_MERGE_MAP.get(category, category)

FILENAME_OVERRIDES = [
    # -------------------------
    # CAREER
    # -------------------------
    (r"(резюме|resume|résumé|cv|curriculum\s+vitae)", "Career/Resume"),

    (r"(стажировк|стаж[её]р|internship|отбор.*стаж|стаж.*отбор)", "Career/Internship"),
    (r"(отбор\s+на\s+стажировк|тз.*стажировк|тестовое.*стажировк)", "Career/Internship"),

    # -------------------------
    # FINANCE
    # -------------------------
    (r"(2\s*ндфл|2ндфл|2[\s_-]*ndfl|3\s*ндфл|3ндфл|3[\s_-]*ndfl)", "Finance"),
    (r"(справк.*доход|налогов.*декларац|налогов.*вычет|кассов.*чек|товарн.*чек)", "Finance"),
    (r"(квитанц.*оплат|сч[её]т.*оплат|плат[её]жн.*поруч|банковск.*выписк)", "Finance"),
    (r"(rzd_ticket|авиабилет|билет|посадочн.*талон)", "Finance"),

    # -------------------------
    # EDUCATION
    # -------------------------
    (r"(справк.*места.*уч[её]б|справк.*обучен|student.*certificate|certificate.*enroll)", "Education/Certificates"),

    (r"(домашн|дз\s*№|дз_|дз-|homework|home\s*assignment)", "Education/Homework"),
    (r"(контрольн.*работ|самостоятельн.*работ|проверочн.*работ)", "Education/Homework"),

    (r"(лабораторн|лаба|лаб[аы]?\s*\d|lab[\s_-]?\d|os[\s_-]?lab|lab[\s_-]?os|physlab)", "Education/Labworks"),
    (r"(отч[её]т.*лаб|лаб.*отч[её]т|физика\s+лаба|лаба\s+физика)", "Education/Labworks"),

    (r"(индивидуальн.*проект|проектн.*работ|курсов.*проект|project\s+report|project\s+presentation)", "Education/Project"),

    (r"(вопросы.*экз|вопросы.*зач[её]т|экзаменационн.*вопрос|экзаменационн.*билет)", "Education/Materials"),
    (r"(конспект|лекц|lecture|lec\d|теорвер|теория|курс\s+лекц|студентам)", "Education/Materials"),
    (r"(учебн.*пособ|методическ.*указ|методическ.*пособ|teacher'?s\s+guide|workbook)", "Education/Materials"),

    # -------------------------
    # IT / LICENSES
    # -------------------------
    (r"(^|[/\\])(ofl|license|licence|copying)(\.[a-z0-9]+)?$", "IT/Licenses"),
    (r"(mit\s+license|apache\s+license|bsd\s+license|gpl|lgpl|open\s+font\s+license)", "IT/Licenses"),
    (r"(copyright|permission\s+is\s+hereby\s+granted|redistribution\s+and\s+use)", "IT/Licenses"),

    # -------------------------
    # MEDIA
    # -------------------------
    (r"(transcript|subtitles|subtitle|субтитр|расшифровк|voice|голосов)", "Media/Transcript"),
    (r"(youtube|youtu\.be|watch\?v=)", "Media/Transcript"),

    (r"(книга|роман|сборник|том\s*\d|isbn|book|novel)", "Media/Books"),
    (r"(история|культура|барокко|романтизм|реализм|модернизм|петергоф|русь)", "Media/Books"),
]


def normalize_text(text: str) -> str:
    return (
        str(text or "")
        .lower()
        .replace("_", " ")
        .replace("-", " ")
        .replace(".", " ")
        .replace("ё", "е")
        .strip()
    )


def normalize_filename(text: str) -> str:
    return (
        str(text or "")
        .lower()
        .replace("ё", "е")
        .strip()
    )


def classify_by_filename(file: dict) -> str | None:
    name = normalize_filename(file.get("name", ""))
    path = normalize_filename(file.get("path", ""))

    filename_text = name

    for pattern, category in FILENAME_OVERRIDES:
        if re.search(pattern, filename_text, flags=re.IGNORECASE):
            return category

    return None


def load_model():
    if not MODEL_PATH.exists():
        raise FileNotFoundError(f"Модель не найдена: {MODEL_PATH}")

    return joblib.load(MODEL_PATH)


def predict_category(text: str, model):
    text = normalize_text(text)

    prediction = model.predict([text])[0]

    confidence = 0.0
    if hasattr(model, "predict_proba"):
        probabilities = model.predict_proba([text])[0]
        confidence = float(max(probabilities))

    return {
        "category": prediction,
        "confidence": confidence,
    }


def classify_file(file: dict, model):
    filename_category = classify_by_filename(file)

    if filename_category:
        return {
            "path": file["path"],
            "name": file["name"],
            "category": merge_category(filename_category),
        }

    name = normalize_text(file.get("name", ""))
    file_text = normalize_text(file.get("text", ""))

    if not file_text:
        path = file.get("path", "")
        ext = file.get("ext", "")
        file_text = normalize_text(extract_preview(path, ext) or "")

    combined_text = f"{name}\n{file_text}".strip()

    if not combined_text:
        combined_text = "document"

    result = predict_category(combined_text, model)

    return {
        "path": file["path"],
        "name": file["name"],
        "category": merge_category(result["category"]),
    }


def main():
    model = load_model()

    with open(FILES_PATH, "r", encoding="utf-8") as f:
        files = json.load(f)

    classified = [classify_file(file, model) for file in files]

    with open(OUTPUT_PATH, "w", encoding="utf-8") as f:
        json.dump(classified, f, ensure_ascii=False, indent=2)

    print(f"Готово. Сохранено в: {OUTPUT_PATH}")


if __name__ == "__main__":
    main()