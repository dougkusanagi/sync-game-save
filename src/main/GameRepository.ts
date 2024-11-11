export class GameRepository {
	all(): Game[] {
		const games: Game[] = [
			{
				name: "Fake",
				exe: "C:/Users/dl_ag/Dev/Js/sync-game-save/tests/sync_files_test/fake_exe_folder/notepad.exe",
				save_files: [
					{ path: "abc.ini", same_dir: true, type: "file" },
					{ path: "aa*", same_dir: true, type: "pattern" },
					{ path: "sub*", same_dir: true, type: "pattern" },
				],
			},
		];

		return games;
	}
}
