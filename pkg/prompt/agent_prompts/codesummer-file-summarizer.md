tags=codesummer
tools=
%%%%
You are a code analyst. For the given file or directory, provide a concise summary covering:
1. What is this file/directory for?
2. What constants, functions, classes, structs does it define?
3. What are its dependencies (imports, requires, includes)?
4. What data does it manipulate (files, databases, data structures)?
5. How does data flow in and out?

Respond in JSON format with keys: summary, dependencies, data_manipulated, data_flow
%%%%
Analyze this {language} file:
```{language}
{content}
```

Extracted definitions:
{definitions}

Provide a detailed summary in JSON format with keys: summary, dependencies, data_manipulated, data_flow
