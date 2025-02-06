// Global state
let currentQuiz = null;
let currentQuestion = 0;
let userAnswers = [];

// DOM Elements
document.addEventListener('DOMContentLoaded', () => {
    const quizContainer = document.getElementById('quiz-container');
    const loginForm = document.getElementById('login-form');
    const registerForm = document.getElementById('register-form');

    // Handle login form submission
    if (loginForm) {
        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(loginForm);
            
            try {
                const response = await fetch('/login', {
                    method: 'POST',
                    body: formData
                });

                if (response.ok) {
                    window.location.href = '/';
                } else {
                    showError('Invalid credentials');
                }
            } catch (error) {
                showError('An error occurred. Please try again.');
            }
        });
    }

    // Quiz navigation functions
    window.nextQuestion = () => {
        if (currentQuestion < currentQuiz.questions.length - 1) {
            currentQuestion++;
            displayQuestion();
        }
    };

    window.previousQuestion = () => {
        if (currentQuestion > 0) {
            currentQuestion--;
            displayQuestion();
        }
    };

    // Handle answer selection
    window.selectAnswer = (answer) => {
        userAnswers[currentQuestion] = answer;
        updateAnswerUI(answer);
    };
});

// Function to load quiz data
async function loadQuiz(quizId) {
    try {
        const response = await fetch(`/api/quiz/${quizId}`);
        if (!response.ok) throw new Error('Quiz not found');
        
        currentQuiz = await response.json();
        currentQuestion = 0;
        userAnswers = new Array(currentQuiz.questions.length).fill(null);
        displayQuestion();
    } catch (error) {
        showError('Failed to load quiz');
    }
}

// Function to display current question
function displayQuestion() {
    const questionContainer = document.getElementById('question-container');
    if (!questionContainer || !currentQuiz) return;

    const question = currentQuiz.questions[currentQuestion];
    
    questionContainer.innerHTML = `
        <div class="question-header">
            <h3>Question ${currentQuestion + 1} of ${currentQuiz.questions.length}</h3>
            <div class="progress-bar">
                <div class="progress" style="width: ${((currentQuestion + 1) / currentQuiz.questions.length) * 100}%"></div>
            </div>
        </div>
        <div class="question-text">
            ${question.text}
        </div>
        <div class="options-container">
            ${question.options.map((option, index) => `
                <button 
                    class="option-btn ${userAnswers[currentQuestion] === option ? 'selected' : ''}"
                    onclick="selectAnswer('${option}')"
                >
                    ${option}
                </button>
            `).join('')}
        </div>
        <div class="navigation-buttons">
            ${currentQuestion > 0 ? 
                '<button onclick="previousQuestion()" class="nav-btn">Previous</button>' : ''}
            ${currentQuestion < currentQuiz.questions.length - 1 ? 
                '<button onclick="nextQuestion()" class="nav-btn">Next</button>' : 
                '<button onclick="submitQuiz()" class="submit-btn">Submit Quiz</button>'}
        </div>
    `;
}

// Function to submit quiz
async function submitQuiz() {
    if (userAnswers.includes(null)) {
        showError('Please answer all questions before submitting');
        return;
    }

    try {
        const response = await fetch('/api/submit-quiz', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                quizId: currentQuiz.id,
                answers: userAnswers
            })
        });

        if (!response.ok) throw new Error('Failed to submit quiz');

        const result = await response.json();
        displayResults(result);
    } catch (error) {
        showError('Failed to submit quiz');
    }
}

// Function to display quiz results
function displayResults(result) {
    const container = document.getElementById('quiz-container');
    if (!container) return;

    container.innerHTML = `
        <div class="results-container">
            <h2>Quiz Results</h2>
            <div class="score-display">
                <div class="score-circle">
                    <span class="score-number">${result.score}%</span>
                </div>
            </div>
            <div class="results-summary">
                <p>Correct Answers: ${result.correctAnswers}</p>
                <p>Total Questions: ${result.totalQuestions}</p>
            </div>
            <button onclick="window.location.href='/'" class="btn-primary">
                Back to Home
            </button>
        </div>
    `;
}

// Utility function to show errors
function showError(message) {
    const errorDiv = document.createElement('div');
    errorDiv.className = 'error-message';
    errorDiv.textContent = message;
    
    document.body.appendChild(errorDiv);
    setTimeout(() => errorDiv.remove(), 3000);
}

// Function to update UI when answer is selected
function updateAnswerUI(selectedAnswer) {
    const options = document.querySelectorAll('.option-btn');
    options.forEach(option => {
        option.classList.remove('selected');
        if (option.textContent.trim() === selectedAnswer) {
            option.classList.add('selected');
        }
    });
} 