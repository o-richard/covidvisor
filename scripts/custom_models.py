from datasets import load_dataset
from transformers import (
    AutoTokenizer,
    AutoModelForSequenceClassification,
    AutoModelForTokenClassification,
    DataCollatorForTokenClassification,
    Trainer,
    TrainingArguments,
)

def classification_model():
    dataset = load_dataset("csv", data_files={"train": "datasets/classification_train.csv", "test": "datasets/classification_test.csv"})
    tokenizer = AutoTokenizer.from_pretrained("google-bert/bert-base-cased")

    def preprocess_data(examples):
        return tokenizer(examples["text"], truncation=True, padding="max_length")
    encoded_dataset = dataset.map(preprocess_data, batched=True)
    
    label_to_id = {"cases_date": 0, "max_cases_duration": 1, "average_cases_duration": 2, "sum_cases_duration": 3, "location_based": 4, "date_based": 5}
    def encode_labels(example):
        example["label"] = label_to_id[example["label"]]
        return example
    encoded_dataset = encoded_dataset.map(encode_labels)

    model = AutoModelForSequenceClassification.from_pretrained("google-bert/bert-base-cased", num_labels=6, torch_dtype="auto")
    training_args = TrainingArguments(output_dir="./classification_model", eval_strategy="epoch")

    trainer = Trainer(
        model=model,
        args=training_args,
        train_dataset=encoded_dataset["train"],
        eval_dataset=encoded_dataset["test"],
    )
    trainer.train()

def ner_model():
    dataset = load_dataset("json", data_files={"train": "datasets/ner_train.json", "test": "datasets/ner_test.json"})
    tokenizer = AutoTokenizer.from_pretrained("dbmdz/bert-large-cased-finetuned-conll03-english")

    label_map = {"CASE_TYPE": 1, "LOCATION": 2, "DATE": 3, "LOWER_BOUND_NUMBER": 4, "DURATION": 5}
    def preprocess_ner_data(examples):
        tokenized_inputs = tokenizer(examples["text"], truncation=True, padding=True, return_offsets_mapping=True)
        labels = []

        for i, entities in enumerate(examples["entities"]):
            word_ids = tokenized_inputs.word_ids(batch_index=i)
            label_ids = [-100] * len(word_ids)
        
            for entity in entities:
                start, end, entity_label = entity["start"], entity["end"], entity["label"]
                for idx, word_idx in enumerate(word_ids):
                    if word_idx is not None:
                        token_start = tokenized_inputs["offset_mapping"][i][idx][0]
                        token_end = tokenized_inputs["offset_mapping"][i][idx][1]
                        
                        if token_start >= start and token_end <= end:
                            label_ids[idx] = label_map[entity_label]
                
            labels.append(label_ids)    
        tokenized_inputs["labels"] = labels
        return tokenized_inputs

    encoded_dataset = dataset.map(preprocess_ner_data, batched=True)

    model = AutoModelForTokenClassification.from_pretrained("dbmdz/bert-large-cased-finetuned-conll03-english", num_labels=6, torch_dtype="auto", ignore_mismatched_sizes=True)
    training_args = TrainingArguments(output_dir="./ner_model", eval_strategy="epoch")
    data_collator = DataCollatorForTokenClassification(tokenizer)
    trainer = Trainer(
        model=model,
        args=training_args,
        train_dataset=encoded_dataset["train"],
        eval_dataset=encoded_dataset["test"],
	    data_collator=data_collator,	
    )
    trainer.train()

classification_model()
ner_model()
