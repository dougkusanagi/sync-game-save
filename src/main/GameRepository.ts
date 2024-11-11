export class GameRepository {
	all(): Game[] {
		const games: Game[] = [
			{
				name: "Fake",
				exe: "D:/fake_game_files/sync_files_test/fake_exe_folder/notepad.exe",
				save_files: [
					{ path: "abc.ini", same_dir: true },
					{ path: "aa*", same_dir: true },
					{ path: "sub*", same_dir: true },
				],
			},
		];

		return games;
	}
}
