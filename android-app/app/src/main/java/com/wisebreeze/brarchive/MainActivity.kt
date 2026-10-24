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
import androidx.compose.ui.res.stringResource
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
    var inputFolder by remember { mutableStateOf<Uri?>(null) }
    var output by remember { mutableStateOf<Uri?>(null) }
    var busy by remember { mutableStateOf(false) }
    var status by remember { mutableStateOf(context.getString(R.string.choose_input)) }
    var error by remember { mutableStateOf(false) }
    val pickInput = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocument()) { input = it; if (it != null) inputFolder = null }
    val pickFolder = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocumentTree()) { inputFolder = it; if (it != null) input = null }
    val pickOutput = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocumentTree()) { output = it }
    MaterialTheme(colorScheme = when { isSystemInDarkTheme() -> darkColorScheme(); Build.VERSION.SDK_INT >= 31 -> dynamicLightColorScheme(context); else -> lightColorScheme() }) {
        Scaffold(topBar = { TopAppBar(title = { Column { Text("Brarchive", fontWeight = FontWeight.SemiBold); Text(stringResource(R.string.app_subtitle), style = MaterialTheme.typography.labelMedium) } }, navigationIcon = { Icon(Icons.Rounded.Archive, null, Modifier.padding(start = 16.dp).size(28.dp)) }) }) { padding ->
            Column(Modifier.fillMaxSize().padding(padding).verticalScroll(rememberScrollState()).padding(20.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
                ElevatedCard { Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(stringResource(R.string.operation), style = MaterialTheme.typography.titleMedium)
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        FilterChip(selected = mode == Mode.DECODE, onClick = { mode = Mode.DECODE }, label = { Text(stringResource(R.string.unpack)) }, leadingIcon = { Icon(Icons.Rounded.LockOpen, null) })
                        FilterChip(selected = mode == Mode.ENCODE, onClick = { mode = Mode.ENCODE }, label = { Text(stringResource(R.string.create_brarchive)) }, leadingIcon = { Icon(Icons.Rounded.Inventory2, null) })
                    }
                } }
                OutlinedCard { Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text(if (mode == Mode.ENCODE) stringResource(R.string.archive_root) else stringResource(R.string.source_archive), style = MaterialTheme.typography.titleMedium)
                    Text(if (mode == Mode.ENCODE) inputFolder?.let { stringResource(R.string.folder_selected) } ?: input?.let { queryName(context, it) } ?: stringResource(R.string.choose_folder_or_file) else input?.let { queryName(context, it) } ?: stringResource(R.string.file_not_selected), style = MaterialTheme.typography.bodyMedium)
                    if (mode == Mode.ENCODE) OutlinedButton(onClick = { pickFolder.launch(null) }, modifier = Modifier.fillMaxWidth(), enabled = !busy) { Icon(Icons.Rounded.FolderOpen, null); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.choose_folder_root)) }
                    OutlinedButton(onClick = { pickInput.launch(arrayOf("*/*")) }, modifier = Modifier.fillMaxWidth(), enabled = !busy) { Icon(Icons.Rounded.FolderOpen, null); Spacer(Modifier.width(8.dp)); Text(if (mode == Mode.ENCODE) stringResource(R.string.choose_single_file) else stringResource(R.string.choose_file)) }
                } }
                OutlinedCard { Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text(stringResource(R.string.destination_folder), style = MaterialTheme.typography.titleMedium)
                    Text(if (output == null) stringResource(R.string.not_selected) else stringResource(R.string.folder_selected), style = MaterialTheme.typography.bodyMedium)
                    OutlinedButton(onClick = { pickOutput.launch(null) }, modifier = Modifier.fillMaxWidth(), enabled = !busy) { Icon(Icons.Rounded.FolderOpen, null); Spacer(Modifier.width(8.dp)); Text(stringResource(R.string.choose_folder)) }
                } }
                Button(onClick = { scope.launch {
                    busy = true; error = false; status = context.getString(R.string.running)
                    try { status = withContext(Dispatchers.IO) { runArchive(context, input, inputFolder, output!!, mode) } }
                    catch (e: Exception) { error = true; status = e.message ?: "Операция не выполнена" }
                    finally { busy = false }
                } }, modifier = Modifier.fillMaxWidth().height(54.dp), enabled = !busy && (input != null || (mode == Mode.ENCODE && inputFolder != null)) && output != null) {
                    if (busy) CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp) else Icon(Icons.Rounded.Archive, null)
                    Spacer(Modifier.width(10.dp)); Text(if (busy) stringResource(R.string.working) else if (mode == Mode.DECODE) stringResource(R.string.unpack_button) else stringResource(R.string.create_archive))
                }
                if (busy) LinearProgressIndicator(Modifier.fillMaxWidth())
                Surface(color = if (error) MaterialTheme.colorScheme.errorContainer else MaterialTheme.colorScheme.secondaryContainer, shape = MaterialTheme.shapes.medium) {
                    Text(status, Modifier.fillMaxWidth().padding(16.dp), color = if (error) MaterialTheme.colorScheme.onErrorContainer else MaterialTheme.colorScheme.onSecondaryContainer)
                }
                Text(stringResource(R.string.local_processing), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
}

private fun runArchive(context: android.content.Context, inputUri: Uri?, inputFolderUri: Uri?, outputTree: Uri, mode: Mode): String {
    val work = File(context.cacheDir, "brarchive-work").apply { deleteRecursively(); mkdirs() }
    val inputFile: File = if (mode == Mode.ENCODE && inputFolderUri != null) {
        val root = File(work, "input-root").apply { mkdirs() }
        val docRoot = DocumentFile.fromTreeUri(context, inputFolderUri) ?: error(context.getString(R.string.cannot_open_folder))
        copyDocumentFolder(context, docRoot, root)
        root
    } else {
        val uri = requireNotNull(inputUri) { "Выберите исходный файл" }
        val file = File(work, safeName(queryName(context, uri)))
        context.contentResolver.openInputStream(uri).use { src -> requireNotNull(src) { context.getString(R.string.cannot_open_input) }; FileOutputStream(file).use { dst -> src.copyTo(dst) } }
        file
    }
    val executable = File(context.applicationInfo.nativeLibraryDir, "libbrarchive.so")
    check(executable.exists() && executable.canExecute()) { context.getString(R.string.engine_missing) }
    val outputFile = File(work, if (mode == Mode.ENCODE) "${if (inputFile.isDirectory) "${inputFolderUri?.lastPathSegment ?: "archive"}" else inputFile.nameWithoutExtension}.brarchive" else "decoded-${inputFile.nameWithoutExtension}")
    val args = if (mode == Mode.ENCODE) listOf("encode", inputFile.absolutePath, "-o", outputFile.absolutePath) else listOf("decode", inputFile.absolutePath, "-o", outputFile.absolutePath)
    val process = ProcessBuilder(listOf(executable.absolutePath) + args).directory(work).redirectErrorStream(true).start()
    val log = process.inputStream.bufferedReader().use { it.readText() }
    val code = process.waitFor()
    check(code == 0) { log.trim().ifBlank { "Архиватор завершился с кодом $code" } }
    check(outputFile.exists()) { "Движок не создал результат. $log" }
    val tree = DocumentFile.fromTreeUri(context, outputTree) ?: error(context.getString(R.string.cannot_open_destination))
    if (outputFile.isDirectory) copyFolderToTree(context, outputFile, tree) else {
        val target = tree.createFile("application/octet-stream", outputFile.name) ?: error(context.getString(R.string.cannot_create_output))
        context.contentResolver.openOutputStream(target.uri, "w").use { dst -> requireNotNull(dst) { context.getString(R.string.cannot_write_output) }; outputFile.inputStream().use { it.copyTo(dst) } }
    }
    return context.getString(R.string.done)
}
private fun copyDocumentFolder(context: android.content.Context, source: DocumentFile, target: File) {
    source.listFiles().forEach { child ->
        val name = child.name ?: "item"
        val out = File(target, name)
        if (child.isDirectory) { out.mkdirs(); copyDocumentFolder(context, child, out) }
        else context.contentResolver.openInputStream(child.uri).use { src -> requireNotNull(src) { context.getString(R.string.cannot_read_file, name) }; FileOutputStream(out).use { dst -> src.copyTo(dst) } }
    }
}
private fun copyFolderToTree(context: android.content.Context, source: File, target: DocumentFile) {
    source.listFiles()?.forEach { child -> if (child.isDirectory) copyFolderToTree(context, child, target.createDirectory(child.name ?: "folder") ?: error(context.getString(R.string.cannot_create_folder))) else {
        val out = target.createFile("application/octet-stream", child.name ?: "file") ?: error("Не удалось создать файл")
        context.contentResolver.openOutputStream(out.uri, "w").use { dst -> requireNotNull(dst); child.inputStream().use { it.copyTo(dst) } }
    } }
}
private fun queryName(context: android.content.Context, uri: Uri): String {
    context.contentResolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME), null, null, null)?.use { if (it.moveToFirst()) return it.getString(0) ?: "archive.brarchive" }
    return uri.lastPathSegment?.substringAfterLast('/') ?: "archive.brarchive"
}
private fun safeName(s: String) = s.replace(Regex("[^A-Za-z0-9._-]"), "_").ifBlank { "input.bin" }
