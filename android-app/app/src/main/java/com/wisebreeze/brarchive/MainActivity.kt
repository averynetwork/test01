package com.wisebreeze.brarchive

import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.provider.OpenableColumns
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Archive
import androidx.compose.material.icons.rounded.FolderOpen
import androidx.compose.material.icons.rounded.LockOpen
import androidx.compose.material.icons.rounded.Inventory2
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.documentfile.provider.DocumentFile
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.io.File
import java.io.FileOutputStream

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) { super.onCreate(savedInstanceState); setContent { BrarchiveApp() } }
}
private enum class Mode { DECODE, ENCODE }

@OptIn(ExperimentalMaterial3Api::class)
@Composable private fun BrarchiveApp() {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var mode by remember { mutableStateOf(Mode.DECODE) }
    var input by remember { mutableStateOf<Uri?>(null) }
    var output by remember { mutableStateOf<Uri?>(null) }
    var busy by remember { mutableStateOf(false) }
    var status by remember { mutableStateOf("Выберите файл и папку назначения") }
    var error by remember { mutableStateOf(false) }
    val pickInput = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocument()) { input = it }
    val pickOutput = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocumentTree()) { output = it }
    MaterialTheme(colorScheme = when { isSystemInDarkTheme() -> darkColorScheme(); Build.VERSION.SDK_INT >= 31 -> dynamicLightColorScheme(context); else -> lightColorScheme() }) {
        Scaffold(topBar = { TopAppBar(title = { Column { Text("Brarchive", fontWeight = FontWeight.SemiBold); Text("Архиватор Bedrock", style = MaterialTheme.typography.labelMedium) } }, navigationIcon = { Icon(Icons.Rounded.Archive, null, Modifier.padding(start = 16.dp).size(28.dp)) }) }) { padding ->
            Column(Modifier.fillMaxSize().padding(padding).verticalScroll(rememberScrollState()).padding(20.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
                ElevatedCard { Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("Операция", style = MaterialTheme.typography.titleMedium)
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        FilterChip(selected = mode == Mode.DECODE, onClick = { mode = Mode.DECODE }, label = { Text("Распаковать") }, leadingIcon = { Icon(Icons.Rounded.LockOpen, null) })
                        FilterChip(selected = mode == Mode.ENCODE, onClick = { mode = Mode.ENCODE }, label = { Text("Создать .brarchive") }, leadingIcon = { Icon(Icons.Rounded.Inventory2, null) })
                    }
                } }
                OutlinedCard { Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text("Исходный файл", style = MaterialTheme.typography.titleMedium)
                    Text(input?.let { queryName(context, it) } ?: "Файл не выбран", style = MaterialTheme.typography.bodyMedium)
                    OutlinedButton(onClick = { pickInput.launch(arrayOf("*/*")) }, modifier = Modifier.fillMaxWidth(), enabled = !busy) { Icon(Icons.Rounded.FolderOpen, null); Spacer(Modifier.width(8.dp)); Text("Выбрать файл") }
                } }
                OutlinedCard { Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text("Папка назначения", style = MaterialTheme.typography.titleMedium)
                    Text(if (output == null) "Не выбрана" else "Папка выбрана", style = MaterialTheme.typography.bodyMedium)
                    OutlinedButton(onClick = { pickOutput.launch(null) }, modifier = Modifier.fillMaxWidth(), enabled = !busy) { Icon(Icons.Rounded.FolderOpen, null); Spacer(Modifier.width(8.dp)); Text("Выбрать папку") }
                } }
                Button(onClick = { scope.launch {
                    busy = true; error = false; status = "Выполняется операция…"
                    try { status = withContext(Dispatchers.IO) { runArchive(context, input!!, output!!, mode) } }
                    catch (e: Exception) { error = true; status = e.message ?: "Операция не выполнена" }
                    finally { busy = false }
                } }, modifier = Modifier.fillMaxWidth().height(54.dp), enabled = !busy && input != null && output != null) {
                    if (busy) CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp) else Icon(Icons.Rounded.Archive, null)
                    Spacer(Modifier.width(10.dp)); Text(if (busy) "Выполняется…" else if (mode == Mode.DECODE) "Распаковать" else "Создать архив")
                }
                if (busy) LinearProgressIndicator(Modifier.fillMaxWidth())
                Surface(color = if (error) MaterialTheme.colorScheme.errorContainer else MaterialTheme.colorScheme.secondaryContainer, shape = MaterialTheme.shapes.medium) {
                    Text(status, Modifier.fillMaxWidth().padding(16.dp), color = if (error) MaterialTheme.colorScheme.onErrorContainer else MaterialTheme.colorScheme.onSecondaryContainer)
                }
                Text("Обработка выполняется локально на устройстве.", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
}

private fun runArchive(context: android.content.Context, inputUri: Uri, outputTree: Uri, mode: Mode): String {
    val work = File(context.cacheDir, "brarchive-work").apply { deleteRecursively(); mkdirs() }
    val inputFile = File(work, safeName(queryName(context, inputUri)))
    context.contentResolver.openInputStream(inputUri).use { src -> requireNotNull(src) { "Не удалось открыть исходный файл" }; FileOutputStream(inputFile).use { dst -> src.copyTo(dst) } }
    val executable = File(context.codeCacheDir, "brarchive-android-arm64")
    if (!executable.exists()) context.assets.open("brarchive-android-arm64").use { src -> FileOutputStream(executable).use { src.copyTo(it) } }
    check(executable.setExecutable(true, true) || executable.canExecute()) { "Не удалось подготовить движок архиватора" }
    val outputFile = File(work, if (mode == Mode.ENCODE) "${inputFile.nameWithoutExtension}.brarchive" else "decoded-${inputFile.nameWithoutExtension}")
    val args = if (mode == Mode.ENCODE) listOf("encode", inputFile.absolutePath, "-o", outputFile.absolutePath) else listOf("decode", inputFile.absolutePath, "-o", outputFile.absolutePath)
    val process = ProcessBuilder(listOf(executable.absolutePath) + args).directory(work).redirectErrorStream(true).start()
    val log = process.inputStream.bufferedReader().use { it.readText() }
    val code = process.waitFor()
    check(code == 0) { log.trim().ifBlank { "Архиватор завершился с кодом $code" } }
    check(outputFile.exists()) { "Движок не создал результат. $log" }
    val tree = DocumentFile.fromTreeUri(context, outputTree) ?: error("Не удалось открыть папку назначения")
    if (outputFile.isDirectory) copyFolderToTree(context, outputFile, tree) else {
        val target = tree.createFile("application/octet-stream", outputFile.name) ?: error("Не удалось создать файл результата")
        context.contentResolver.openOutputStream(target.uri, "w").use { dst -> requireNotNull(dst) { "Не удалось записать результат" }; outputFile.inputStream().use { it.copyTo(dst) } }
    }
    return "Готово. Результат сохранён в выбранную папку."
}
private fun copyFolderToTree(context: android.content.Context, source: File, target: DocumentFile) {
    source.listFiles()?.forEach { child -> if (child.isDirectory) copyFolderToTree(context, child, target.createDirectory(child.name ?: "folder") ?: error("Не удалось создать папку")) else {
        val out = target.createFile("application/octet-stream", child.name ?: "file") ?: error("Не удалось создать файл")
        context.contentResolver.openOutputStream(out.uri, "w").use { dst -> requireNotNull(dst); child.inputStream().use { it.copyTo(dst) } }
    } }
}
private fun queryName(context: android.content.Context, uri: Uri): String {
    context.contentResolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME), null, null, null)?.use { if (it.moveToFirst()) return it.getString(0) ?: "archive.brarchive" }
    return uri.lastPathSegment?.substringAfterLast('/') ?: "archive.brarchive"
}
private fun safeName(s: String) = s.replace(Regex("[^A-Za-z0-9._-]"), "_").ifBlank { "input.bin" }
