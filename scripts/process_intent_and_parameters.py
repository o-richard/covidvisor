import json
import re
import sys

from transformers import AutoTokenizer, AutoModelForSequenceClassification, AutoModelForTokenClassification, pipeline
from word2number import w2n

classification_model = AutoModelForSequenceClassification.from_pretrained("./scripts/classification_model/checkpoint-375")
classification_model_tokenizer = AutoTokenizer.from_pretrained("google-bert/bert-base-cased")
classification_pipeline = pipeline("text-classification", model=classification_model, tokenizer=classification_model_tokenizer, top_k=1)
classification_mapping = {
    "LABEL_0": "cases_date",
    "LABEL_1": "max_cases_duration",
    "LABEL_2": "average_cases_duration",
    "LABEL_3": "sum_cases_duration",
    "LABEL_4": "location_based",
    "LABEL_5": "date_based",
}

ner_model = AutoModelForTokenClassification.from_pretrained("./scripts/ner_model/checkpoint-375")
ner_model_tokenizer = AutoTokenizer.from_pretrained("dbmdz/bert-large-cased-finetuned-conll03-english")
ner_pipeline = pipeline("ner", model=ner_model, tokenizer=ner_model_tokenizer, aggregation_strategy="simple")
ner_mapping = {
    "LABEL_1": "CASE_TYPE",
    "LABEL_2": "LOCATION",
    "LABEL_3": "DATE",
    "LABEL_4": "LOWER_BOUND_NUMBER",
    "LABEL_5": "DURATION",
}

def get_integer_from_text(text, default=1):
    match = re.search(r'\d+', text)
    if match:
        return str(match.group())
    
    try:
        return f"{w2n.word_to_num(text)}"
    except ValueError:
        return "1"

def cleanup_entities(intent, entities):
    location = entities.get("LOCATION", "")
    
    case_type = entities.get("CASE_TYPE", "active_cases")
    if "death" in case_type:
        case_type = "death_cases"
    elif "recover" in case_type:
        case_type = "recovery_cases"
    else:
        case_type = "active_cases"

    duration = entities.get("DURATION", entities.get("DATE", "all_time"))
    duration_count = get_integer_from_text(duration, 1)
    if "day" in duration:
        duration = f"- {duration_count} days"
    elif "week" in duration:
        duration = f"- {duration_count} weeks"
    elif "month" in duration:
        duration = f"- {duration_count} months"
    elif "year" in duration:
        duration = f"- {duration_count} years"
    else:
        duration = "all_time"

    if intent == "cases_date":
        return {"location": location, "date": "today", "case_type": case_type}
    if intent == "max_cases_duration" or intent == "average_cases_duration" or intent == "sum_cases_duration":
        return {"location": location, "case_type": case_type, "duration": duration}
    if intent == "location_based":
        return {"case_type": case_type}
    if intent == "date_based":
        return {"location": location, "case_type": case_type, "duration": duration, "lower_bound_number": get_integer_from_text(entities["LOWER_BOUND_NUMBER"], 1)}
    
    return {}


def process_query(query):
    try:
        classification = classification_pipeline(query)[0][0]
        intent = classification_mapping.get(classification['label'], "no_match")
        if classification['score'] < 0.8:
            intent = "intent"
    except IndexError:
        intent = "no_match"
    
    ner_results = ner_pipeline(query)
    entities = {}

    for entity in ner_results:
        label_type = ner_mapping.get(entity["entity_group"], None)
        if label_type is None or entity["score"] < 0.8:
            continue
        if label_type not in entities or entity["score"] > entities[label_type]["score"]:
            entities[label_type] = {"name": entity["word"], "score": entity["score"]}
    
    for key in entities.keys():
        entities[key] = entities[key]["name"]
    
    output = json.dumps({"intent": intent, "entities": cleanup_entities(intent, entities)})
    return output 

for line in sys.stdin:
    if "q" == line.rstrip():
	    break

    answer = process_query(line)
    print(answer, end="", flush=True)
