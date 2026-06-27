def normalize_text(text: str) -> str:
    return (
        str(text or "")
        .lower()
        .replace("_", " ")
        .replace("-", " ")
        .replace(".", " ")
        .strip()
    )


def normalize_filename(text: str) -> str:
    return (
        str(text or "")
        .lower()
        .strip()
    )
