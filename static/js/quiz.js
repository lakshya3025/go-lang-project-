let currentQuestionIndex = 0;
let score = 0;
let timer = 60;
let timerInterval;
const answers = [];
let quizId = window.location.pathname.split('/').pop();

function startTimer() {
    timerInterval = setInterval(() => {
        timer--;
        document.getElementById('time').textContent = timer;
        
        if (timer <= 10) {
            document.getElementById('time').classList.add('warning');
        }
        
        if (timer <= 0) {
            clearInterval(timerInterval);
            submitQuiz();
        }
    }, 1000);
}

function updateScore(isCorrect) {
    if (isCorrect) {
        // Bonus points for quick answers
        const timeBonus = Math.floor(timer / 10);
        const points = 100 + timeBonus;
        score += points;
        
        showFloatingPoints(`+${points}`);
    }
    document.getElementById('score').textContent = score;
}

function showFloatingPoints(text) {
    const points = document.createElement('div');
    points.className = 'floating-points';
    points.textContent = text;
    document.querySelector('.score-display').appendChild(points);
    
    setTimeout(() => points.remove(), 1000);
}

function selectAnswer(button) {
    // Remove selection from other buttons in the same question
    const questionCard = button.closest('.question-card');
    questionCard.querySelectorAll('.option-btn').forEach(btn => {
        btn.classList.remove('selected');
    });

    // Select this button
    button.classList.add('selected');

    // Enable next/submit button
    const nextBtn = questionCard.querySelector('.next-btn, .submit-btn');
    if (nextBtn) nextBtn.disabled = false;

    // Store the answer
    answers[currentQuestionIndex] = button.dataset.option;
    
    // Show immediate feedback
    const isCorrect = button.dataset.option === questions[currentQuestionIndex].correctAnswer;
    updateScore(isCorrect);
    
    button.classList.add(isCorrect ? 'correct' : 'incorrect');
}

function showQuestion(index) {
    document.querySelectorAll('.question-card').forEach(card => {
        card.style.display = 'none';
    });
    document.querySelector(`[data-question-index="${index}"]`).style.display = 'block';
    document.getElementById('current-question').textContent = index + 1;
    currentQuestionIndex = index;
}

function previousQuestion() {
    if (currentQuestionIndex > 0) {
        showQuestion(currentQuestionIndex - 1);
    }
}

function nextQuestion() {
    const totalQuestions = document.querySelectorAll('.question-card').length;
    if (currentQuestionIndex < totalQuestions - 1) {
        showQuestion(currentQuestionIndex + 1);
    }
}

async function submitQuiz() {
    try {
        const response = await fetch('/api/submit-quiz', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                quizId: parseInt(quizId),
                answers: answers
            })
        });

        if (!response.ok) {
            const error = await response.text();
            throw new Error(error || 'Failed to submit quiz');
        }

        const result = await response.json();
        showResults(result);
    } catch (error) {
        console.error('Error submitting quiz:', error);
        alert('Error submitting quiz: ' + error.message);
    }
}

function showResults(result) {
    const quizContainer = document.getElementById('quiz-container');
    const resultHtml = `
        <div class="results-card">
            <h2>Quiz Results</h2>
            <div class="score-display">
                <div class="score">${Math.round(result.score)}%</div>
                <div class="score-details">
                    <p>Correct Answers: ${result.correctAnswers}</p>
                    <p>Total Questions: ${result.totalQuestions}</p>
                </div>
            </div>
            
            <div class="answers-review">
                <h3>Review Your Answers</h3>
                ${result.questions.map((q, index) => `
                    <div class="answer-item ${q.isCorrect ? 'correct' : 'incorrect'}">
                        <div class="question-text">${q.text}</div>
                        <div class="answer-details">
                            <p>Your Answer: <span class="${q.isCorrect ? 'correct-text' : 'incorrect-text'}">${q.userAnswer}</span></p>
                            ${!q.isCorrect ? `<p>Correct Answer: <span class="correct-text">${q.correctAnswer}</span></p>` : ''}
                        </div>
                        ${q.explanation ? `<div class="answer-explanation">${q.explanation}</div>` : ''}
                    </div>
                `).join('')}
            </div>

            <div class="result-actions">
                <a href="/" class="btn-primary">Back to Home</a>
                <button onclick="location.reload()" class="btn-secondary">Try Again</button>
            </div>
        </div>
    `;
    quizContainer.innerHTML = resultHtml;
}

// Add click handlers to all option buttons
document.querySelectorAll('.option-btn').forEach(button => {
    button.addEventListener('click', () => selectAnswer(button));
});

// Add context panel toggle
document.querySelectorAll('.context-toggle').forEach(toggle => {
    toggle.addEventListener('click', () => {
        const content = toggle.nextElementSibling;
        content.classList.toggle('hidden');
        toggle.textContent = content.classList.contains('hidden') ? 
            '📚 Show Context' : '📚 Hide Context';
    });
});

// Start the game
startTimer(); 