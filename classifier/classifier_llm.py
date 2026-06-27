import sys
import json
import normalizer
from ollama import chat
from pathlib import Path
from typing import Literal
from pydantic import BaseModel, ValidationError

PROJECT_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(PROJECT_ROOT))


FILES_PATH = PROJECT_ROOT / "files" / "files.json"
OUTPUT_PATH = PROJECT_ROOT / "files" / "classified_files.json"

RETRIES = 4

class Answer(BaseModel):
    answer: Literal["Education/Assignments", 
                    "Documents/Personal",  
                    "Education/Materials",
                    "Career/Internship",
                    "Career/Resume", 
                    "Education/Project", 
                    "Finance", 
                    "IT/Licenses", 
                    "Media/Transcript"
                    ]
    
SCHEMA = Answer.model_json_schema()

def classify_file(file: dict):
    name = normalizer.normalize_text(file.get("name", ""))
    file_text = normalizer.normalize_text(file.get("text", ""))

    messages = [
        {
            "role": "system",
            "content": (
                "You are a file classifier. "
                "Classify the file by its meaning using ONLY the file name and text. "
                "Return JSON only. No markdown. No explanation. "
                "The JSON must match exactly this shape: "
                '{"answer": "<one allowed category>"} '
                "\n\n"
                "Allowed categories and meanings:\n"
                "\n"
                "1. Education/Assignments\n"
                "Use for homework, exercises, tasks, lab work, quizzes, tests, submissions, "
                "course assignments, grading rubrics, and files that look like student work to complete.\n"
                "\n"
                "2. Documents/Personal\n"
                "Use for personal documents, identity documents, letters, applications, certificates, "
                "personal notes, forms, scans, and files about a person's private life or administration.\n"
                "\n"
                "3. Education/Materials\n"
                "Use for study materials, lecture notes, slides, textbooks, articles, tutorials, "
                "course references, learning resources, and educational explanations not meant as a task to submit.\n"
                "\n"
                "4. Career/Internship\n"
                "Use for internships, job applications, vacancies, job descriptions, company programs, "
                "interview preparation, recruitment, cover letters, and internship-related documents.\n"
                "\n"
                "5. Career/Resume\n"
                "Use for resumes, CVs, portfolios, professional profiles, biographies, skill summaries, "
                "and files describing a person's work experience or qualifications.\n"
                "\n"
                "6. Education/Project\n"
                "Use for school or university projects, research projects, project reports, project plans, "
                "presentations, source materials for a project, and teamwork/project deliverables.\n"
                "\n"
                "7. Finance\n"
                "Use for invoices, receipts, payments, bank documents, taxes, budgets, salaries, prices, "
                "financial statements, purchases, subscriptions, and money-related files.\n"
                "\n"
                "8. IT/Licenses\n"
                "Use for software licenses, activation keys, certificates, API keys, tokens, credentials, "
                "software subscriptions, technical access documents, and license agreements.\n"
                "\n"
                "9. Media/Transcript\n"
                "Use for transcripts, subtitles, captions, interview transcripts, meeting transcripts, "
                "video/audio text dumps, speech-to-text output, and dialogue scripts.\n"
                "\n\n"
                "Decision rules:\n"
                "- If the file is an assignment to complete or submit, choose Education/Assignments.\n"
                "- If the file teaches or explains something, choose Education/Materials.\n"
                "- If the file is about a project deliverable, choose Education/Project.\n"
                "- If the file is mainly a CV/resume, choose Career/Resume.\n"
                "- If the file is about applying to a job or internship, choose Career/Internship.\n"
                "- If the file is about money, choose Finance.\n"
                "- If the file contains software license or access information, choose IT/Licenses.\n"
                "- If the file is mostly spoken text converted to written text, choose Media/Transcript.\n"
                "- If none of the above fits clearly, choose Documents/Personal.\n"
                "\n\n"
                "Allowed JSON answers:\n"
                '{"answer": "Education/Assignments"}\n'
                '{"answer": "Documents/Personal"}\n'
                '{"answer": "Education/Materials"}\n'
                '{"answer": "Career/Internship"}\n'
                '{"answer": "Career/Resume"}\n'
                '{"answer": "Education/Project"}\n'
                '{"answer": "Finance"}\n'
                '{"answer": "IT/Licenses"}\n'
                '{"answer": "Media/Transcript"}'
            )
        },
        {
            "role": "user",
            "content": f"{name}\n{file_text}"
        }
    ]

    options = {
        "temperature": 0,
        "num_predict": 20
    }

    for _ in range(0, RETRIES-1):
        response = chat(
            model="qwen3:1.7b",
            messages=messages,
            options=options,
            format=SCHEMA
        )

        content = response.message.content

        try: 
            parsed = Answer.model_validate_json(content)
            return {
                "name": name,
                "path": file["path"],
                "category":parsed.answer
                }
        
        except ValidationError:
            messages.append({
                "role": "user",
                "content": (
                    "Your previous response was invalid. "
                    "Return EXACTLY valid JSON matching this schema: "
                    f"{SCHEMA}"
                ),
            })

    return {
                "name": name,
                "path": file["path"],
                "category": "Education/Assignments"
            }



def main():
    with open(FILES_PATH, "r", encoding="utf-8") as f:
        files = json.load(f)

    classified = [classify_file(file) for file in files]

    with open(OUTPUT_PATH, "w", encoding="utf-8") as f:
        json.dump(classified, f, ensure_ascii=False, indent=2)

    print(f"Готово. Сохранено в: {OUTPUT_PATH}")


if __name__ == "__main__":
    main()