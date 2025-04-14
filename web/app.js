async function loadSeasons() {
    try {
        const response = await fetch('/api/seasons');
        const seasons = await response.json();

        const seasonSelect = document.getElementById('seasonSelect');
        seasonSelect.innerHTML = '';

        seasons.forEach(season => {
            const option = document.createElement('option');
            option.value = season.value; // ← Исходный сезон (2023)
            option.textContent = season.label; // ← Отформатированный (2023/24)
            seasonSelect.appendChild(option);
        });

        currentSeason = seasons[0].value || '';
        seasonSelect.value = currentSeason;
    } catch (error) {
        console.error("Ошибка загрузки сезонов:", error);
    }
}
async function loadLeagues() {
    try {
        const response = await fetch('/api/leagues');
        const leagues = await response.json();

        const leaguesGrid = document.getElementById('leaguesGrid');
        leaguesGrid.innerHTML = '';

        leagues.forEach(league => {
            const card = document.createElement('div');
            card.className = 'col-6 col-md-2 mb-3';

            card.innerHTML = `
                <div class="card h-100" style="cursor: pointer;" onclick="window.location.href='/league?id=${league.id}&season=${currentSeason}'">
                    <div class="d-flex justify-content-center align-items-center" style="height: 100px;">
                        <img src="${league.icon}" class="card-img-top img-fluid" alt="${league.name}" style="max-height: 120px; max-width: 120px;">
                    </div>
                    <div class="card-body text-center">
                        <h6 class="card-title">${league.name}</h6>
                        <p class="card-text text-muted">${league.country}</p>
                    </div>
                </div>
            `;

            leaguesGrid.appendChild(card);
        });
    } catch (error) {
        console.error("Ошибка загрузки лиг:", error);
    }
}

async function loadTopTeams() {
    try {
        const response = await fetch(`/api/top-teams?season=${currentSeason}`);
        const data = await response.json();


        const topTeamsTable = document.querySelector('#topTeamsTable tbody');
        if (!data.teams || data.teams.length === 0) {
            topTeamsTable.innerHTML = '<tr><td colspan="5">No data available</td></tr>';
            return;
        }

        topTeamsTable.innerHTML = data.teams.map((team, index) => `
            <tr>
                <td>${index + 1}</td>
                <td>
                    <img src="https://media.api-sports.io/football/teams/${team.id}.png" 
                         class="team-icon me-2" 
                         style="width: 24px; height: 24px;"
                         onerror="this.style.display='none'">
                    ${team.name || 'N/A'}
                </td>
                <td>${team.avg_points ? team.avg_points.toFixed(2) : '0.00'}</td>
                <td>${team.avg_goals_for ? team.avg_goals_for.toFixed(2) : '0.00'}</td>
                <td>${team.avg_goals_against ? team.avg_goals_against.toFixed(2) : '0.00'}</td>
            </tr>
        `).join('');
    } catch (error) {
        console.error("Ошибка загрузки данных:", error);
        topTeamsTable.innerHTML = '<tr><td colspan="5">Error loading data</td></tr>';
    }
}
async function loadTopPlayers() {
    try {
        const response = await fetch(`/api/top-players?season=${currentSeason}`);
        const players = await response.json();

        const topPlayersTable = document.querySelector('#topPlayersTable tbody');
        if (!players || players.length === 0) {
            topPlayersTable.innerHTML = '<tr><td colspan="7">No data available</td></tr>';
            return;
        }

        topPlayersTable.innerHTML = players.map((player, index) => `
            <tr>
                <td>${index + 1}</td>
                <td>
                    <img src="https://media.api-sports.io/football/teams/${player.team_id}.png" 
                         class="team-icon me-2" 
                         style="width: 24px; height: 24px;"
                         onerror="this.style.display='none'">
                         ${player.name}
                </td>
                
                <td>${player.total_points}</td>
                <td>${player.total_goals}</td>
                <td>${player.total_assists}</td>
                <td>${player.avg_points ? player.avg_points.toFixed(2) : 'N/A'}</td>
                <td>${player.avg_goal ? player.avg_goal.toFixed(2) : 'N/A'}</td>
            </tr>
        `).join('');
    } catch (error) {
        console.error("Ошибка загрузки данных:", error);
        topPlayersTable.innerHTML = '<tr><td colspan="7">Error loading data</td></tr>';
    }
}
document.addEventListener('DOMContentLoaded', async () => {
    await loadSeasons();

    loadLeagues();
    loadTopTeams();
    loadTopPlayers();

    document.getElementById('seasonSelect').addEventListener('change', (e) => {
        currentSeason = e.target.value;
        loadTopTeams();
    });
});