import requests
import json

# First API call with reasoning
response = requests.post(
  url="https://openrouter.ai/api/v1/chat/completions",
  headers={
    "Authorization": "Bearer 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",
    "Content-Type": "application/json",
  },
  data=json.dumps({
    "model": "google/gemma-3-27b-it:free",
    "messages": [
        {
          "role": "user",
          "content": "How many r's are in the word 'strawberry'?"
        }
      ],
    "reasoning": {"enabled": False}
  })
)

# Extract the assistant message with reasoning_details
response = response.json()
print("First API response:", response)  # Debugging output
response = response['choices'][0]['message']

# Preserve the assistant message with reasoning_details
messages = [
  {"role": "user", "content": "How many r's are in the word 'strawberry'?"},
  {
    "role": "assistant",
    "content": response.get('content'),
    "reasoning_details": response.get('reasoning_details')  # Pass back unmodified
  },
  {"role": "user", "content": "Are you sure? Think carefully."}
]

# Second API call - model continues reasoning from where it left off
response2 = requests.post(
  url="https://openrouter.ai/api/v1/chat/completions",
  data=json.dumps({
    "model": "google/gemma-3-27b-it:free",
    "messages": messages,  # Includes preserved reasoning_details
    "reasoning": {"enabled": False}
  })
)