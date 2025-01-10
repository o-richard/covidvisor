# Workflow

- [x] Capture audio input.
- [x] Convert audio to text.
- [x] Send text to local Hugging Face Models.
- [x] Retrieve and process the intent and parameters.
- [x] Query the SQLite database.
- [x] Convert the result to audio output.
- [x] Play the response to the user.

# Dependencies

- Linux
- `alsa-utils` - **arecord** (version 1.2.13) is used for voice recording from the microphone. Other alternative is PortAudio (cross-compatible)
- `espeak-ng`- **espeak** (version 1.52) is used for text to speech. Other alternatives are festival.
- `Assembly.AI account` - **assembly.ai** is used for speech to text. Other alternatives are Google Speech to Text.
- Python - A virtual environment should be created. The directory is `venv`. Install requirements from requirements.txt on the virtual environment. The first time calling `scripts/main.py` could take a while since some resources need to be download - subsequent calls are faster. The models in `scripts/custom_models.py` should be trained prior. Based on the results, the classification_model & ner_model paths can be updated in `scripts/process_intent_and_parameters`
- Go - All dependencies should be downloaded. Data should be seed into the database on the initial build/run.

# Data source

- [Covid Cases](https://www.isibang.ac.in/~incovid19/dataopen/summarymohfw1update.csv)
- [Covid Management Protocol](https://covid19dashboard.mohfw.gov.in/pdf/UpdatedDetailedClinicalManagementProtocolforCOVID19adultsdated24052021.pdf). ChatGPT is used to summarize the document.

# Query Types

## Case-Based Queries

- Cases by Date: "How many active cases are in Karnataka today?"
- Max Cases: "What’s the highest number of recoveries in Delhi this week?"
- Average Cases: "Show me the average deaths in Maharashtra over the past 30 days."
- Total Cases: "How many cases were reported in Goa last month?"

## Trend-Based Queries

- Location-Based: "Which state has the highest number of active cases?"
- Date-Based: "When did Tamil Nadu cross 1,000 active cases for the first time?"

## Freeform Questions

Ask anything related to COVID-19 data

# Resources

- [ASCII Generator](https://www.patorjk.com/software/taag/#p=display&f=Colossal&t=COVIDVisor)