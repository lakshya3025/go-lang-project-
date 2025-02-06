// Add error handling function
function showError(message) {
    const errorDiv = document.createElement('div');
    errorDiv.className = 'error-message';
    errorDiv.textContent = message;
    document.body.appendChild(errorDiv);
    setTimeout(() => errorDiv.remove(), 3000);
}

document.getElementById('quiz-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const formData = {
        title: document.getElementById('title').value,
        category: parseInt(document.getElementById('category').value),
        difficulty: document.getElementById('difficulty').value,
        questionCount: parseInt(document.getElementById('questionCount').value)
    };

    try {
        const response = await fetch('/admin/create-quiz', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(formData)
        });

        if (!response.ok) {
            throw new Error('Failed to create quiz');
        }

        const data = await response.json();
        window.location.href = `/quiz/${data.quizId}`;
    } catch (error) {
        alert('Error creating quiz: ' + error.message);
    }
}); 